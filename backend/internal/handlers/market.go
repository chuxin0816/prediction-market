package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"gorm.io/gorm"
)

type MarketHandler struct {
	db *gorm.DB
}

func NewMarketHandler(db *gorm.DB) *MarketHandler {
	return &MarketHandler{db: db}
}

func (h *MarketHandler) List(c *gin.Context) {
	var markets []models.Market

	query := h.db.Model(&models.Market{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&markets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, markets)
}

func (h *MarketHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	var market models.Market
	if err := h.db.First(&market, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, market)
}

func (h *MarketHandler) GetTrades(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid market id"})
		return
	}

	var trades []models.Trade
	if err := h.db.Where("market_id = ?", id).Order("created_at DESC").Limit(100).Find(&trades).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trades)
}
