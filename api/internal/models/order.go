package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderSide is bid or ask.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType is limit or market.
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderStatus in persistence.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusMatched   OrderStatus = "matched"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
)

// Order is the persisted order entity.
type Order struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	IdempotencyKey string         `gorm:"uniqueIndex;size:64" json:"-"`
	UserID         string         `gorm:"not null;index" json:"user_id"`
	MarketID       string         `gorm:"not null;index" json:"market_id"`
	Side           OrderSide      `gorm:"not null" json:"side"`
	Type           OrderType      `gorm:"not null" json:"type"`
	Price          string         `gorm:"type:numeric" json:"price"`
	Size           string         `gorm:"type:numeric;not null" json:"size"`
	Remaining      string         `gorm:"type:numeric" json:"remaining"`
	Status         OrderStatus    `gorm:"not null;default:pending" json:"status"`
	CancelFee      string         `gorm:"type:numeric;default:0" json:"cancel_fee"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides table name.
func (Order) TableName() string {
	return "orders"
}

// BeforeCreate sets UUID.
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
