package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

type MarketStatus string

const (
	MarketStatusPending   MarketStatus = "pending"
	MarketStatusActive    MarketStatus = "active"
	MarketStatusResolved  MarketStatus = "resolved"
	MarketStatusCancelled MarketStatus = "cancelled"
)

type Market struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	ChainID         *uint64        `json:"chain_id"`
	Question        string         `gorm:"not null" json:"question"`
	Description     string         `json:"description"`
	Outcomes        datatypes.JSON `gorm:"not null" json:"outcomes"`
	EndTime         time.Time      `gorm:"not null" json:"end_time"`
	ResolutionTime  time.Time      `gorm:"not null" json:"resolution_time"`
	ResolvedOutcome *uint8         `json:"resolved_outcome"`
	Status          MarketStatus   `gorm:"not null;default:pending" json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type MarketWithStats struct {
	Market
	TotalVolume decimal.Decimal `json:"total_volume"`
	LastPrice   decimal.Decimal `json:"last_price"`
}
