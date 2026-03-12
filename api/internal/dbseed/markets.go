package dbseed

import (
	"time"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"gorm.io/gorm"
)

// SeedInitialMarkets inserts a default set of markets if none exist yet.
// This is only for local/dev so that UIs like /markets have something to show.
func SeedInitialMarkets(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Market{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now().UTC()
	markets := []models.Market{
		{
			ID:            "BTC-USDC",
			BaseAsset:     "BTC",
			QuoteAsset:    "USDC",
			TickSize:      "0.10",
			MinSize:       "0.0001",
			FeeRate:       "0.001",
			CancelFeeRate: "0.0005",
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "ETH-USDC",
			BaseAsset:     "ETH",
			QuoteAsset:    "USDC",
			TickSize:      "0.05",
			MinSize:       "0.001",
			FeeRate:       "0.001",
			CancelFeeRate: "0.0005",
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "SOL-USDC",
			BaseAsset:     "SOL",
			QuoteAsset:    "USDC",
			TickSize:      "0.01",
			MinSize:       "0.1",
			FeeRate:       "0.001",
			CancelFeeRate: "0.0005",
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "ARB-USDC",
			BaseAsset:     "ARB",
			QuoteAsset:    "USDC",
			TickSize:      "0.0001",
			MinSize:       "1",
			FeeRate:       "0.0015",
			CancelFeeRate: "0.00075",
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "OP-USDC",
			BaseAsset:     "OP",
			QuoteAsset:    "USDC",
			TickSize:      "0.0001",
			MinSize:       "1",
			FeeRate:       "0.0015",
			CancelFeeRate: "0.00075",
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	return db.Create(&markets).Error
}

