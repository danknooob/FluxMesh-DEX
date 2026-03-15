package repository

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"gorm.io/gorm"
)

type BalanceRepository struct {
	db *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

func (r *BalanceRepository) ListByUser(ctx context.Context, userID string) ([]models.Balance, error) {
	var out []models.Balance
	err := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_list_balances_by_user($1)", userID).
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	// Ensure callers get a non-nil slice so JSON encodes as [] not null.
	if out == nil {
		out = []models.Balance{}
	}
	return out, nil
}
