package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

type CreateMarketRequest struct {
	Question       string    `json:"question" binding:"required"`
	Description    string    `json:"description"`
	Outcomes       []string  `json:"outcomes" binding:"required,min=2"`
	EndTime        time.Time `json:"end_time" binding:"required"`
	ResolutionTime time.Time `json:"resolution_time" binding:"required"`
}

func (h *AdminHandler) CreateMarket(c *gin.Context) {
	isAdmin, _ := c.Get("admin")
	if isAdmin != true {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req CreateMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.EndTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end time must be in the future"})
		return
	}

	if req.ResolutionTime.Before(req.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resolution time must be after end time"})
		return
	}

	outcomesJSON, err := json.Marshal(req.Outcomes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process outcomes"})
		return
	}

	market := models.Market{
		Question:       req.Question,
		Description:    req.Description,
		Outcomes:       datatypes.JSON(outcomesJSON),
		EndTime:        req.EndTime,
		ResolutionTime: req.ResolutionTime,
		Status:         models.MarketStatusActive,
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	if err := tx.Create(&market).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Initialize AMM liquidity pool with equal reserves
	numOutcomes := len(req.Outcomes)
	initialReserve := "100"
	reserveStrings := make([]string, numOutcomes)
	for i := range reserveStrings {
		reserveStrings[i] = initialReserve
	}
	reservesJSON, err := json.Marshal(reserveStrings)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal reserves"})
		return
	}

	// k = 100^numOutcomes
	k := "1"
	hundredDec, _ := strconv.ParseInt(initialReserve, 10, 64)
	kVal := int64(1)
	for i := 0; i < numOutcomes; i++ {
		kVal *= hundredDec
	}
	k = strconv.FormatInt(kVal, 10)

	pool := models.LiquidityPool{
		MarketID:        market.ID,
		OutcomeReserves: datatypes.JSON(reservesJSON),
		K:               k,
	}
	if err := tx.Create(&pool).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create liquidity pool: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, market)
}

type ResolveMarketRequest struct {
	Outcome uint8 `json:"outcome" binding:"required"`
}

func (h *AdminHandler) ResolveMarket(c *gin.Context) {
	isAdmin, _ := c.Get("admin")
	if isAdmin != true {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	marketID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	var req ResolveMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var market models.Market
	if err := h.db.First(&market, marketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
		return
	}

	if market.Status != models.MarketStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "market is not active"})
		return
	}

	var outcomes []string
	if err := json.Unmarshal(market.Outcomes, &outcomes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "corrupted market data"})
		return
	}

	if int(req.Outcome) < 1 || int(req.Outcome) > len(outcomes) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid outcome"})
		return
	}

	market.ResolvedOutcome = &req.Outcome
	market.Status = models.MarketStatusResolved

	if err := h.db.Save(&market).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, market)
}
