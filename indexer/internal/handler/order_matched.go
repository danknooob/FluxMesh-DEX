package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/model"
	"github.com/danknooob/fluxmesh-dex/indexer/internal/repository"
)

// OrderMatchedHandler processes orders.matched events.
// Uses a single atomic stored function to update both order
// statuses and create the trade record in one transaction.
type OrderMatchedHandler struct {
	orders repository.OrderWriter
}

func NewOrderMatchedHandler(o repository.OrderWriter) *OrderMatchedHandler {
	return &OrderMatchedHandler{orders: o}
}

func (h *OrderMatchedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt model.OrderMatchedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("unmarshal orders.matched: %w", err)
	}

	req := repository.MatchRequest{
		MakerOrderID:   evt.MakerOrderID,
		TakerOrderID:   evt.TakerOrderID,
		MakerRemaining: evt.MakerRemaining,
		TakerRemaining: evt.TakerRemaining,
		TradeID:        evt.TradeID,
		MarketID:       evt.MarketID,
		Price:          evt.Price,
		Size:           evt.Size,
		MakerSide:      evt.MakerSide,
	}
	if err := h.orders.ProcessMatch(ctx, req); err != nil {
		return fmt.Errorf("process match trade %s: %w", evt.TradeID, err)
	}

	log.Printf("indexer: matched trade %s (%s %s @ %s)", evt.TradeID, evt.MarketID, evt.Size, evt.Price)
	return nil
}
