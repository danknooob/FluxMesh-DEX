package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/model"
	"github.com/danknooob/fluxmesh-dex/indexer/internal/repository"
)

// BalanceUpdatedHandler processes balances.updated events.
// Responsibility: upsert the user's asset balance row in Postgres.
type BalanceUpdatedHandler struct {
	balances repository.BalanceWriter
}

func NewBalanceUpdatedHandler(b repository.BalanceWriter) *BalanceUpdatedHandler {
	return &BalanceUpdatedHandler{balances: b}
}

func (h *BalanceUpdatedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt model.BalanceUpdatedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("unmarshal balances.updated: %w", err)
	}

	if evt.UserID == "" || evt.Asset == "" {
		log.Printf("indexer: skipping balances.updated with empty user_id or asset")
		return nil
	}

	if err := h.balances.Upsert(ctx, evt.UserID, evt.Asset, evt.Available, evt.Locked); err != nil {
		return fmt.Errorf("upsert balance user=%s asset=%s: %w", evt.UserID, evt.Asset, err)
	}

	log.Printf("indexer: balance updated user=%s asset=%s avail=%s locked=%s",
		evt.UserID, evt.Asset, evt.Available, evt.Locked)
	return nil
}
