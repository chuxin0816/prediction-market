package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Trade struct {
	ID           uint64          `gorm:"primaryKey" json:"id"`
	MarketID     uint64          `gorm:"not null;index" json:"market_id"`
	MakerOrderID uint64          `gorm:"not null" json:"maker_order_id"`
	TakerOrderID uint64          `gorm:"not null" json:"taker_order_id"`
	MakerAddress string          `gorm:"not null;size:42" json:"maker_address"`
	TakerAddress string          `gorm:"not null;size:42" json:"taker_address"`
	Outcome      uint8           `gorm:"not null" json:"outcome"`
	Price        decimal.Decimal `gorm:"not null;type:decimal(10,4)" json:"price"`
	Quantity     decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"quantity"`
	ChainSettled bool            `gorm:"default:false" json:"chain_settled"`
	CreatedAt    time.Time       `json:"created_at"`
}
