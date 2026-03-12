package repository

import (
	"context"

	"gorm.io/gorm"
)

type gormBalanceWriter struct {
	db *gorm.DB
}

func NewBalanceWriter(db *gorm.DB) BalanceWriter {
	return &gormBalanceWriter{db: db}
}

func (r *gormBalanceWriter) Upsert(ctx context.Context, userID, asset, available, locked string) error {
	return r.db.WithContext(ctx).
		Exec("SELECT fn_upsert_balance($1,$2,$3,$4)", userID, asset, available, locked).Error
}
