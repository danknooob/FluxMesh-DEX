package repository

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"gorm.io/gorm"
)

// MarketRepository handles market read model.
type MarketRepository struct {
	db *gorm.DB
}

// NewMarketRepository creates a MarketRepository.
func NewMarketRepository(db *gorm.DB) *MarketRepository {
	return &MarketRepository{db: db}
}

// List returns all enabled markets.
func (r *MarketRepository) List(ctx context.Context) ([]models.Market, error) {
	var list []models.Market
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&list).Error
	return list, err
}

// GetByID returns a market by ID.
func (r *MarketRepository) GetByID(ctx context.Context, id string) (*models.Market, error) {
	var m models.Market
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}
