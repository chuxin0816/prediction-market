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

	outcomesJSON, _ := json.Marshal(req.Outcomes)

	market := models.Market{
		Question:       req.Question,
		Description:    req.Description,
		Outcomes:       datatypes.JSON(outcomesJSON),
		EndTime:        req.EndTime,
		ResolutionTime: req.ResolutionTime,
		Status:         models.MarketStatusActive,
	}

	if err := h.db.Create(&market).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, market)
}

type ResolveMarketRequest struct {
	Outcome uint8 `json:"outcome" binding:"required"`
}

func (h *AdminHandler) ResolveMarket(c *gin.Context) {
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
	json.Unmarshal(market.Outcomes, &outcomes)

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
