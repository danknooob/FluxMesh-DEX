package model

import (
	"time"

	"gorm.io/gorm"
)

// Trade is the persisted trade record, populated by the indexer
// when it consumes orders.matched and trades.settled events.
type Trade struct {
	ID           string         `gorm:"primaryKey;size:64" json:"id"`
	MarketID     string         `gorm:"not null;index" json:"market_id"`
	MakerOrderID string         `gorm:"not null;index" json:"maker_order_id"`
	TakerOrderID string         `gorm:"not null;index" json:"taker_order_id"`
	Price        string         `gorm:"type:numeric;not null" json:"price"`
	Size         string         `gorm:"type:numeric;not null" json:"size"`
	MakerSide    string         `gorm:"not null" json:"maker_side"`
	SettledAt    *time.Time     `json:"settled_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Trade) TableName() string {
	return "trades"
}
