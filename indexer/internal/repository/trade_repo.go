package repository

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/model"
	"gorm.io/gorm"
)

type gormTradeWriter struct {
	db *gorm.DB
}

func NewTradeWriter(db *gorm.DB) TradeWriter {
	return &gormTradeWriter{db: db}
}

func (r *gormTradeWriter) Create(ctx context.Context, trade *model.Trade) error {
	return r.db.WithContext(ctx).
		Exec(
			"SELECT fn_create_trade_if_not_exists($1,$2,$3,$4,$5,$6,$7)",
			trade.ID, trade.MarketID,
			trade.MakerOrderID, trade.TakerOrderID,
			trade.Price, trade.Size, trade.MakerSide,
		).Error
}

func (r *gormTradeWriter) MarkSettled(ctx context.Context, tradeID string) error {
	return r.db.WithContext(ctx).
		Exec("SELECT fn_mark_trade_settled($1)", tradeID).Error
}
