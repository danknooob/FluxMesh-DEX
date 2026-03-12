package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/model"
	"github.com/danknooob/fluxmesh-dex/indexer/internal/repository"
)

// OrderRejectedHandler processes orders.rejected events.
// Responsibility: mark the order as rejected in Postgres.
type OrderRejectedHandler struct {
	orders repository.OrderWriter
}

func NewOrderRejectedHandler(o repository.OrderWriter) *OrderRejectedHandler {
	return &OrderRejectedHandler{orders: o}
}

func (h *OrderRejectedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt model.OrderRejectedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("unmarshal orders.rejected: %w", err)
	}

	if err := h.orders.UpdateStatus(ctx, evt.OrderID, "rejected", ""); err != nil {
		return fmt.Errorf("reject order %s: %w", evt.OrderID, err)
	}

	log.Printf("indexer: rejected order %s reason=%s", evt.OrderID, evt.Reason)
	return nil
}
