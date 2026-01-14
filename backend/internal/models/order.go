package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderSide string
type OrderStatus string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"

	OrderStatusOpen      OrderStatus = "open"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID             uint64          `gorm:"primaryKey" json:"id"`
	MarketID       uint64          `gorm:"not null;index" json:"market_id"`
	UserAddress    string          `gorm:"not null;size:42;index" json:"user_address"`
	Outcome        uint8           `gorm:"not null" json:"outcome"`
	Side           OrderSide       `gorm:"not null;size:4" json:"side"`
	Price          decimal.Decimal `gorm:"not null;type:decimal(10,4)" json:"price"`
	Quantity       decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"quantity"`
	FilledQuantity decimal.Decimal `gorm:"not null;type:decimal(20,6);default:0" json:"filled_quantity"`
	Status         OrderStatus     `gorm:"not null;size:20;default:open" json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (o *Order) RemainingQuantity() decimal.Decimal {
	return o.Quantity.Sub(o.FilledQuantity)
}
