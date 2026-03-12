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

func (r *gormOrderWriter) ProcessMatch(ctx context.Context, req MatchRequest) error {
	return r.db.WithContext(ctx).
		Exec(
			"SELECT fn_process_order_matched($1,$2,$3,$4,$5,$6,$7,$8,$9)",
			req.MakerOrderID, req.TakerOrderID,
			req.MakerRemaining, req.TakerRemaining,
			req.TradeID, req.MarketID,
			req.Price, req.Size, req.MakerSide,
		).Error
}
