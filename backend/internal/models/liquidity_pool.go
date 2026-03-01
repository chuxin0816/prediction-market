package models

import (
	"time"

	"gorm.io/datatypes"
)

type LiquidityPool struct {
	MarketID        uint64         `gorm:"primaryKey" json:"market_id"`
	OutcomeReserves datatypes.JSON `gorm:"not null" json:"outcome_reserves"`
	K               string         `gorm:"not null;type:text" json:"k"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
