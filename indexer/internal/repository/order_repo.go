package repository

import (
	"context"

	"gorm.io/gorm"
)

type gormOrderWriter struct {
	db *gorm.DB
}

func NewOrderWriter(db *gorm.DB) OrderWriter {
	return &gormOrderWriter{db: db}
}

func (r *gormOrderWriter) UpdateStatus(ctx context.Context, orderID, status, remaining string) error {
	return r.db.WithContext(ctx).
		Exec("SELECT fn_update_order_status($1,$2,$3)", orderID, status, remaining).Error
}
