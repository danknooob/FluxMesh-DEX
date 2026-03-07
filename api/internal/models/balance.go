package models

import (
	"time"

	"gorm.io/gorm"
)

// Balance is per-user, per-asset (read model).
type Balance struct {
	UserID    string         `gorm:"primaryKey" json:"user_id"`
	Asset     string         `gorm:"primaryKey" json:"asset"`
	Available string         `gorm:"type:numeric;not null" json:"available"`
	Locked    string         `gorm:"type:numeric;default:0" json:"locked"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides table name.
func (Balance) TableName() string {
	return "balances"
}
