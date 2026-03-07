package models

import (
	"time"

	"gorm.io/gorm"
)

// Market represents a trading pair (read model / API view).
type Market struct {
	ID         string         `gorm:"primaryKey" json:"id"`
	BaseAsset  string         `gorm:"not null" json:"base_asset"`
	QuoteAsset string         `gorm:"not null" json:"quote_asset"`
	TickSize   string         `gorm:"type:numeric" json:"tick_size"`
	MinSize    string         `gorm:"type:numeric" json:"min_size"`
	FeeRate    string         `gorm:"type:numeric" json:"fee_rate"`
	Enabled    bool           `gorm:"default:true" json:"enabled"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides table name.
func (Market) TableName() string {
	return "markets"
}
