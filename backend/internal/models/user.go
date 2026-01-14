package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type UserBalance struct {
	UserAddress string          `gorm:"primaryKey;size:42" json:"user_address"`
	Available   decimal.Decimal `gorm:"not null;type:decimal(20,6);default:0" json:"available"`
	Locked      decimal.Decimal `gorm:"not null;type:decimal(20,6);default:0" json:"locked"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type BalanceLog struct {
	ID           uint64          `gorm:"primaryKey" json:"id"`
	UserAddress  string          `gorm:"not null;size:42;index" json:"user_address"`
	ChangeType   string          `gorm:"not null;size:20" json:"change_type"`
	Amount       decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"amount"`
	BalanceAfter decimal.Decimal `gorm:"not null;type:decimal(20,6)" json:"balance_after"`
	ReferenceID  *uint64         `json:"reference_id"`
	CreatedAt    time.Time       `json:"created_at"`
}
