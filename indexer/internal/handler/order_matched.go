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
// Responsibilities: update maker/taker order statuses and create a trade record.
type OrderMatchedHandler struct {
	orders repository.OrderWriter
	trades repository.TradeWriter
}

func NewOrderMatchedHandler(o repository.OrderWriter, t repository.TradeWriter) *OrderMatchedHandler {
	return &OrderMatchedHandler{orders: o, trades: t}
}

func (h *OrderMatchedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt model.OrderMatchedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("unmarshal orders.matched: %w", err)
	}

	if err := h.orders.UpdateStatus(ctx, evt.MakerOrderID, "matched", ""); err != nil {
		log.Printf("indexer: update maker order %s: %v", evt.MakerOrderID, err)
	}
	if err := h.orders.UpdateStatus(ctx, evt.TakerOrderID, "matched", ""); err != nil {
		log.Printf("indexer: update taker order %s: %v", evt.TakerOrderID, err)
	}

	trade := &model.Trade{
		ID:           evt.TradeID,
		MarketID:     evt.MarketID,
		MakerOrderID: evt.MakerOrderID,
		TakerOrderID: evt.TakerOrderID,
		Price:        evt.Price,
		Size:         evt.Size,
		MakerSide:    evt.MakerSide,
	}
	if err := h.trades.Create(ctx, trade); err != nil {
		return fmt.Errorf("create trade %s: %w", evt.TradeID, err)
	}

	log.Printf("indexer: matched trade %s (%s %s @ %s)", evt.TradeID, evt.MarketID, evt.Size, evt.Price)
	return nil
}
