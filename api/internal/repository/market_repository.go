package repository

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"gorm.io/gorm"
)

// MarketRepository handles market reads via stored functions.
type MarketRepository struct {
	db *gorm.DB
}

func NewMarketRepository(db *gorm.DB) *MarketRepository {
	return &MarketRepository{db: db}
}

func (r *MarketRepository) List(ctx context.Context) ([]models.Market, error) {
	var list []models.Market
	err := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_list_enabled_markets()").
		Scan(&list).Error
	return list, err
}

func (r *MarketRepository) GetByID(ctx context.Context, id string) (*models.Market, error) {
	var m models.Market
	result := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_get_market_by_id($1)", id).
		Scan(&m)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &m, nil
}
