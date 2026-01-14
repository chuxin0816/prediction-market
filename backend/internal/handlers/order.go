package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"github.com/prediction-market/backend/internal/services/orderbook"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type OrderHandler struct {
	db  *gorm.DB
	obm *orderbook.OrderBookManager
}

func NewOrderHandler(db *gorm.DB, obm *orderbook.OrderBookManager) *OrderHandler {
	return &OrderHandler{db: db, obm: obm}
}

type PlaceOrderRequest struct {
	MarketID uint64          `json:"market_id" binding:"required"`
	Outcome  uint8           `json:"outcome" binding:"required"`
	Side     string          `json:"side" binding:"required,oneof=buy sell"`
	Price    decimal.Decimal `json:"price" binding:"required"`
	Quantity decimal.Decimal `json:"quantity" binding:"required"`
}

type PlaceOrderResponse struct {
	Order  *models.Order   `json:"order"`
	Trades []models.Trade  `json:"trades"`
}

type OrderBookResponse struct {
	Buys  []PriceLevelResponse `json:"buys"`
	Sells []PriceLevelResponse `json:"sells"`
}

type PriceLevelResponse struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	userAddress, exists := c.Get("user_address")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate price between 0.01 and 0.99
	minPrice := decimal.NewFromFloat(0.01)
	maxPrice := decimal.NewFromFloat(0.99)
	if req.Price.LessThan(minPrice) || req.Price.GreaterThan(maxPrice) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be between 0.01 and 0.99"})
		return
	}

	// Check market exists and is active
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

	userAddr := userAddress.(string)
	side := models.OrderSide(req.Side)

	// Calculate required balance for buy orders (price * quantity)
	requiredBalance := decimal.Zero
	if side == models.OrderSideBuy {
		requiredBalance = req.Price.Mul(req.Quantity)

		// Check user balance
		var balance models.UserBalance
		if err := h.db.First(&balance, "user_address = ?", userAddr).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if balance.Available.LessThan(requiredBalance) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
			return
		}
	}

	// Create order with status Open
	order := &models.Order{
		MarketID:       req.MarketID,
		UserAddress:    userAddr,
		Outcome:        req.Outcome,
		Side:           side,
		Price:          req.Price,
		Quantity:       req.Quantity,
		FilledQuantity: decimal.Zero,
		Status:         models.OrderStatusOpen,
	}

	// Start DB transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	// Lock balance for buy orders (move from Available to Locked)
	if side == models.OrderSideBuy {
		result := tx.Model(&models.UserBalance{}).
			Where("user_address = ? AND available >= ?", userAddr, requiredBalance).
			Updates(map[string]interface{}{
				"available": gorm.Expr("available - ?", requiredBalance),
				"locked":    gorm.Expr("locked + ?", requiredBalance),
			})
		if result.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
		if result.RowsAffected == 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
			return
		}
	}

	// Save order to DB
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add order to orderbook
	ob := h.obm.GetOrCreate(req.MarketID, req.Outcome)
	matchResult, err := ob.AddOrder(order)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add order to orderbook: " + err.Error()})
		return
	}

	// Save trades from MatchResult
	for i := range matchResult.Trades {
		matchResult.Trades[i].TakerOrderID = order.ID
		if err := tx.Create(&matchResult.Trades[i]).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Update maker orders status
	for _, makerOrder := range matchResult.MakerOrders {
		if err := tx.Model(&models.Order{}).
			Where("id = ?", makerOrder.ID).
			Updates(map[string]interface{}{
				"filled_quantity": makerOrder.FilledQuantity,
				"status":          makerOrder.Status,
			}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Update taker order status
	if err := tx.Model(&models.Order{}).
		Where("id = ?", order.ID).
		Updates(map[string]interface{}{
			"filled_quantity": order.FilledQuantity,
			"status":          order.Status,
		}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PlaceOrderResponse{
		Order:  order,
		Trades: matchResult.Trades,
	})
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userAddress, exists := c.Get("user_address")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	userAddr := userAddress.(string)

	// Get order and verify ownership
	var order models.Order
	if err := h.db.First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Verify order belongs to user
	if order.UserAddress != userAddr {
		c.JSON(http.StatusForbidden, gin.H{"error": "order does not belong to user"})
		return
	}

	// Check order is Open or Partial
	if order.Status != models.OrderStatusOpen && order.Status != models.OrderStatusPartial {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order cannot be cancelled"})
		return
	}

	// Remove from orderbook
	ob := h.obm.GetOrCreate(order.MarketID, order.Outcome)
	ob.RemoveOrder(&order)

	// Calculate amount to unlock (remaining quantity * price)
	unlockAmount := order.RemainingQuantity().Mul(order.Price)

	// Start transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	// Unlock balance for buy orders
	if order.Side == models.OrderSideBuy && unlockAmount.GreaterThan(decimal.Zero) {
		result := tx.Model(&models.UserBalance{}).
			Where("user_address = ?", userAddr).
			Updates(map[string]interface{}{
				"available": gorm.Expr("available + ?", unlockAmount),
				"locked":    gorm.Expr("locked - ?", unlockAmount),
			})
		if result.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
	}

	// Set status to Cancelled
	order.Status = models.OrderStatusCancelled
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userAddress, exists := c.Get("user_address")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userAddr := userAddress.(string)

	var orders []models.Order
	query := h.db.Where("user_address = ?", userAddr)

	// Optionally filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Limit(100).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) GetOrderBook(c *gin.Context) {
	marketID, err := strconv.ParseUint(c.Param("market_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	// Get outcome from query param (default 1)
	outcome := uint8(1)
	if outcomeStr := c.Query("outcome"); outcomeStr != "" {
		outcomeVal, err := strconv.ParseUint(outcomeStr, 10, 8)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid outcome"})
			return
		}
		outcome = uint8(outcomeVal)
	}

	depth := h.obm.GetDepth(marketID, outcome)

	response := OrderBookResponse{
		Buys:  make([]PriceLevelResponse, 0),
		Sells: make([]PriceLevelResponse, 0),
	}

	if depth != nil {
		for _, level := range depth.Buys {
			response.Buys = append(response.Buys, PriceLevelResponse{
				Price:    level.Price,
				Quantity: level.Quantity,
			})
		}
		for _, level := range depth.Sells {
			response.Sells = append(response.Sells, PriceLevelResponse{
				Price:    level.Price,
				Quantity: level.Quantity,
			})
		}
	}

	c.JSON(http.StatusOK, response)
}
