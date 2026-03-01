package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"github.com/prediction-market/backend/internal/services/amm"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AMMHandler struct {
	db *gorm.DB
}

func NewAMMHandler(db *gorm.DB) *AMMHandler {
	return &AMMHandler{db: db}
}

// getOrCreatePool gets the pool for a market, creating one if it doesn't exist (for legacy markets).
func (h *AMMHandler) getOrCreatePool(tx *gorm.DB, marketID uint64, lockForUpdate bool) (models.LiquidityPool, error) {
	var pool models.LiquidityPool
	query := tx
	if lockForUpdate {
		query = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&pool, "market_id = ?", marketID).Error
	if err == nil {
		return pool, nil
	}
	if err != gorm.ErrRecordNotFound {
		return pool, err
	}

	// Pool doesn't exist — create one for this legacy market
	var market models.Market
	if err := tx.First(&market, marketID).Error; err != nil {
		return pool, fmt.Errorf("market not found")
	}
	var outcomes []string
	if err := json.Unmarshal(market.Outcomes, &outcomes); err != nil {
		return pool, fmt.Errorf("corrupted market data")
	}

	numOutcomes := len(outcomes)
	reserveStrings := make([]string, numOutcomes)
	for i := range reserveStrings {
		reserveStrings[i] = "100"
	}
	reservesJSON, _ := json.Marshal(reserveStrings)

	kVal := int64(1)
	for i := 0; i < numOutcomes; i++ {
		kVal *= 100
	}

	pool = models.LiquidityPool{
		MarketID:        marketID,
		OutcomeReserves: datatypes.JSON(reservesJSON),
		K:               strconv.FormatInt(kVal, 10),
	}
	if err := tx.Create(&pool).Error; err != nil {
		return pool, err
	}

	// Re-lock if needed
	if lockForUpdate {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&pool, "market_id = ?", marketID).Error; err != nil {
			return pool, err
		}
	}
	return pool, nil
}

type BuyRequest struct {
	MarketID uint64 `json:"market_id" binding:"required"`
	Outcome  uint8  `json:"outcome" binding:"required"`
	Amount   string `json:"amount" binding:"required"`
}

type SellRequest struct {
	MarketID uint64 `json:"market_id" binding:"required"`
	Outcome  uint8  `json:"outcome" binding:"required"`
	Shares   string `json:"shares" binding:"required"`
}

type TradeResponse struct {
	SharesReceived decimal.Decimal   `json:"shares_received,omitempty"`
	USDCReceived   decimal.Decimal   `json:"usdc_received,omitempty"`
	AvgPrice       decimal.Decimal   `json:"avg_price"`
	NewPrices      []decimal.Decimal `json:"new_prices"`
}

type PricesResponse struct {
	Prices       []decimal.Decimal `json:"prices"`
	PoolReserves []string          `json:"pool_reserves"`
}

type QuoteResponse struct {
	SharesOut   decimal.Decimal   `json:"shares_out,omitempty"`
	USDCOut     decimal.Decimal   `json:"usdc_out,omitempty"`
	AvgPrice    decimal.Decimal   `json:"avg_price"`
	PriceImpact decimal.Decimal   `json:"price_impact"`
	NewPrices   []decimal.Decimal `json:"new_prices"`
}

func (h *AMMHandler) Buy(c *gin.Context) {
	userAddress, ok := c.Get("user_address")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userAddr, ok := userAddress.(string)
	if !ok || userAddr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user address"})
		return
	}

	var req BuyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usdcAmount, err := decimal.NewFromString(req.Amount)
	if err != nil || usdcAmount.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	// Validate market exists and is active
	var market models.Market
	if err := h.db.First(&market, req.MarketID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if market.Status != models.MarketStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market is not active"})
		return
	}

	// Validate outcome index
	var outcomes []string
	if err := json.Unmarshal(market.Outcomes, &outcomes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "corrupted market data"})
		return
	}
	outcomeIndex := int(req.Outcome) - 1 // 1-based to 0-based
	if outcomeIndex < 0 || outcomeIndex >= len(outcomes) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid outcome"})
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	// Lock and check user balance
	var balance models.UserBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		FirstOrCreate(&balance, models.UserBalance{UserAddress: userAddr}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock balance"})
		return
	}

	if balance.Available.LessThan(usdcAmount) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}

	// Lock the pool row (auto-create for legacy markets)
	pool, err := h.getOrCreatePool(tx, req.MarketID, true)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "liquidity pool not found: " + err.Error()})
		return
	}

	// Calculate buy
	sharesOut, newReserves, err := amm.CalculateBuy(&pool, outcomeIndex, usdcAmount)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update pool reserves
	reservesJSON, err := amm.MarshalReserves(newReserves)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal reserves"})
		return
	}
	pool.OutcomeReserves = datatypes.JSON(reservesJSON)
	if err := tx.Save(&pool).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Deduct USDC from user
	balance.Available = balance.Available.Sub(usdcAmount)
	if err := tx.Save(&balance).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create/update user position
	var position models.UserPosition
	result := tx.Where("user_address = ? AND market_id = ? AND outcome = ?",
		userAddr, req.MarketID, req.Outcome).First(&position)
	if result.Error == gorm.ErrRecordNotFound {
		position = models.UserPosition{
			UserAddress: userAddr,
			MarketID:    req.MarketID,
			Outcome:     req.Outcome,
			Shares:      sharesOut,
		}
		if err := tx.Create(&position).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	} else {
		position.Shares = position.Shares.Add(sharesOut)
		if err := tx.Save(&position).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Record trade
	avgPrice := usdcAmount.Div(sharesOut)
	trade := models.Trade{
		MarketID:     req.MarketID,
		MakerOrderID: 0,
		TakerOrderID: 0,
		MakerAddress: "amm",
		TakerAddress: userAddr,
		Outcome:      req.Outcome,
		Price:        avgPrice,
		Quantity:     sharesOut,
	}
	if err := tx.Create(&trade).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log balance change
	balanceLog := models.BalanceLog{
		UserAddress:  userAddr,
		ChangeType:   "amm_buy",
		Amount:       usdcAmount.Neg(),
		BalanceAfter: balance.Available,
		ReferenceID:  &trade.ID,
	}
	if err := tx.Create(&balanceLog).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get new prices
	newPrices, _ := amm.GetPrices(&pool)

	c.JSON(http.StatusOK, TradeResponse{
		SharesReceived: sharesOut,
		AvgPrice:       avgPrice,
		NewPrices:      newPrices,
	})
}

func (h *AMMHandler) Sell(c *gin.Context) {
	userAddress, ok := c.Get("user_address")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userAddr, ok := userAddress.(string)
	if !ok || userAddr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user address"})
		return
	}

	var req SellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sharesIn, err := decimal.NewFromString(req.Shares)
	if err != nil || sharesIn.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shares amount"})
		return
	}

	// Validate market
	var market models.Market
	if err := h.db.First(&market, req.MarketID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if market.Status != models.MarketStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market is not active"})
		return
	}

	// Validate outcome
	var outcomes []string
	if err := json.Unmarshal(market.Outcomes, &outcomes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "corrupted market data"})
		return
	}
	outcomeIndex := int(req.Outcome) - 1
	if outcomeIndex < 0 || outcomeIndex >= len(outcomes) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid outcome"})
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	// Check user has enough shares
	var position models.UserPosition
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_address = ? AND market_id = ? AND outcome = ?",
			userAddr, req.MarketID, req.Outcome).
		First(&position).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no position found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if position.Shares.LessThan(sharesIn) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient shares"})
		return
	}

	// Lock the pool (auto-create for legacy markets)
	pool, err := h.getOrCreatePool(tx, req.MarketID, true)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "liquidity pool not found: " + err.Error()})
		return
	}

	// Calculate sell
	usdcOut, newReserves, err := amm.CalculateSell(&pool, outcomeIndex, sharesIn)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update pool reserves
	reservesJSON, err := amm.MarshalReserves(newReserves)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal reserves"})
		return
	}
	pool.OutcomeReserves = datatypes.JSON(reservesJSON)
	if err := tx.Save(&pool).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Deduct shares from user position
	position.Shares = position.Shares.Sub(sharesIn)
	if err := tx.Save(&position).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add USDC to user balance
	var balance models.UserBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		FirstOrCreate(&balance, models.UserBalance{UserAddress: userAddr}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock balance"})
		return
	}
	balance.Available = balance.Available.Add(usdcOut)
	if err := tx.Save(&balance).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Record trade
	avgPrice := usdcOut.Div(sharesIn)
	trade := models.Trade{
		MarketID:     req.MarketID,
		MakerOrderID: 0,
		TakerOrderID: 0,
		MakerAddress: userAddr,
		TakerAddress: "amm",
		Outcome:      req.Outcome,
		Price:        avgPrice,
		Quantity:     sharesIn,
	}
	if err := tx.Create(&trade).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log balance change
	balanceLog := models.BalanceLog{
		UserAddress:  userAddr,
		ChangeType:   "amm_sell",
		Amount:       usdcOut,
		BalanceAfter: balance.Available,
		ReferenceID:  &trade.ID,
	}
	if err := tx.Create(&balanceLog).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get new prices
	newPrices, _ := amm.GetPrices(&pool)

	c.JSON(http.StatusOK, TradeResponse{
		USDCReceived: usdcOut,
		AvgPrice:     avgPrice,
		NewPrices:    newPrices,
	})
}

func (h *AMMHandler) GetPrices(c *gin.Context) {
	marketIDStr := c.Query("market_id")
	if marketIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market_id is required"})
		return
	}
	marketID, err := strconv.ParseUint(marketIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market_id"})
		return
	}

	pool, err := h.getOrCreatePool(h.db, marketID, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity pool not found: " + err.Error()})
		return
	}

	prices, err := amm.GetPrices(&pool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var reserveStrings []string
	if err := json.Unmarshal(pool.OutcomeReserves, &reserveStrings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse reserves"})
		return
	}

	c.JSON(http.StatusOK, PricesResponse{
		Prices:       prices,
		PoolReserves: reserveStrings,
	})
}

func (h *AMMHandler) GetQuote(c *gin.Context) {
	marketIDStr := c.Query("market_id")
	if marketIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market_id is required"})
		return
	}
	marketID, err := strconv.ParseUint(marketIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market_id"})
		return
	}

	outcomeStr := c.Query("outcome")
	if outcomeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "outcome is required"})
		return
	}
	outcome, err := strconv.ParseUint(outcomeStr, 10, 8)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid outcome"})
		return
	}
	outcomeIndex := int(outcome) - 1 // 1-based to 0-based

	amountStr := c.Query("amount")
	if amountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount is required"})
		return
	}
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	side := c.Query("side")
	if side != "buy" && side != "sell" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side must be 'buy' or 'sell'"})
		return
	}

	pool, err := h.getOrCreatePool(h.db, marketID, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity pool not found: " + err.Error()})
		return
	}

	// Get current prices for price impact calculation
	currentPrices, err := amm.GetPrices(&pool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if outcomeIndex < 0 || outcomeIndex >= len(currentPrices) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid outcome"})
		return
	}

	if side == "buy" {
		sharesOut, newReserves, err := amm.CalculateBuy(&pool, outcomeIndex, amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		avgPrice := amount.Div(sharesOut)
		priceImpact := avgPrice.Sub(currentPrices[outcomeIndex]).Div(currentPrices[outcomeIndex])

		// Get new prices after trade
		newReservesJSON, _ := amm.MarshalReserves(newReserves)
		newPool := models.LiquidityPool{OutcomeReserves: datatypes.JSON(newReservesJSON)}
		newPrices, _ := amm.GetPrices(&newPool)

		c.JSON(http.StatusOK, QuoteResponse{
			SharesOut:   sharesOut,
			AvgPrice:    avgPrice,
			PriceImpact: priceImpact,
			NewPrices:   newPrices,
		})
	} else {
		usdcOut, newReserves, err := amm.CalculateSell(&pool, outcomeIndex, amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		avgPrice := usdcOut.Div(amount)
		priceImpact := currentPrices[outcomeIndex].Sub(avgPrice).Div(currentPrices[outcomeIndex])

		// Get new prices after trade
		newReservesJSON, _ := amm.MarshalReserves(newReserves)
		newPool := models.LiquidityPool{OutcomeReserves: datatypes.JSON(newReservesJSON)}
		newPrices, _ := amm.GetPrices(&newPool)

		c.JSON(http.StatusOK, QuoteResponse{
			USDCOut:     usdcOut,
			AvgPrice:    avgPrice,
			PriceImpact: priceImpact,
			NewPrices:   newPrices,
		})
	}
}

func (h *AMMHandler) GetPositions(c *gin.Context) {
	userAddress, ok := c.Get("user_address")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userAddr, ok := userAddress.(string)
	if !ok || userAddr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user address"})
		return
	}

	var positions []models.UserPosition
	query := h.db.Where("user_address = ?", userAddr)

	if marketIDStr := c.Query("market_id"); marketIDStr != "" {
		marketID, err := strconv.ParseUint(marketIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market_id"})
			return
		}
		query = query.Where("market_id = ?", marketID)
	}

	if err := query.Find(&positions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, positions)
}
