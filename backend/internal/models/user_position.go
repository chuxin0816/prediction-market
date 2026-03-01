package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type UserPosition struct {
	ID          uint64          `gorm:"primaryKey" json:"id"`
	UserAddress string          `gorm:"not null;size:42;uniqueIndex:idx_user_market_outcome" json:"user_address"`
	MarketID    uint64          `gorm:"not null;uniqueIndex:idx_user_market_outcome" json:"market_id"`
	Outcome     uint8           `gorm:"not null;uniqueIndex:idx_user_market_outcome" json:"outcome"`
	Shares      decimal.Decimal `gorm:"not null;type:decimal(20,6);default:0" json:"shares"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
