package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/model"
	"github.com/danknooob/fluxmesh-dex/indexer/internal/repository"
)

// TradeSettledHandler processes trades.settled events.
// Responsibility: stamp the settled_at timestamp on the trade row.
type TradeSettledHandler struct {
	trades repository.TradeWriter
}

func NewTradeSettledHandler(t repository.TradeWriter) *TradeSettledHandler {
	return &TradeSettledHandler{trades: t}
}

func (h *TradeSettledHandler) Handle(ctx context.Context, payload []byte) error {
	var evt model.TradeSettledEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("unmarshal trades.settled: %w", err)
	}

	if err := h.trades.MarkSettled(ctx, evt.TradeID); err != nil {
		return fmt.Errorf("settle trade %s: %w", evt.TradeID, err)
	}

	log.Printf("indexer: settled trade %s", evt.TradeID)
	return nil
}
