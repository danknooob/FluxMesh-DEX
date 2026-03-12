package repository

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/model"
)

// OrderWriter updates order statuses in Postgres.
// Segregated from read concerns — the indexer only writes.
type OrderWriter interface {
	UpdateStatus(ctx context.Context, orderID string, status string, remaining string) error
}

// TradeWriter creates and updates trade records.
type TradeWriter interface {
	Create(ctx context.Context, trade *model.Trade) error
	MarkSettled(ctx context.Context, tradeID string) error
}

// BalanceWriter upserts per-user per-asset balances.
type BalanceWriter interface {
	Upsert(ctx context.Context, userID, asset, available, locked string) error
}
