package engine

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
)

type MatchedTrade struct {
	TradeID      string `json:"trade_id"`
	MarketID     string `json:"market_id"`
	MakerOrderID string `json:"maker_order_id"`
	TakerOrderID string `json:"taker_order_id"`
	Price        string `json:"price"`
	Size         string `json:"size"`
	MakerSide    string `json:"maker_side"`
	Ts           string `json:"ts"`
}

type SettlementBatch struct {
	Trades []MatchedTrade
}

type EventProducer interface {
	PublishTradesSettled(ctx context.Context, payload interface{}) error
	PublishBalancesUpdated(ctx context.Context, payload interface{}) error
}

type KafkaProducer struct {
	tradesWriter   *kafka.Writer
	balancesWriter *kafka.Writer
}

func NewKafkaProducer(brokers []string, tradesTopic, balancesTopic string) *KafkaProducer {
	addr := kafka.TCP(brokers...)
	return &KafkaProducer{
		tradesWriter: &kafka.Writer{
			Addr:     addr,
			Topic:    tradesTopic,
			Balancer: &kafka.LeastBytes{},
		},
		balancesWriter: &kafka.Writer{
			Addr:     addr,
			Topic:    balancesTopic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *KafkaProducer) PublishTradesSettled(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.tradesWriter.WriteMessages(ctx, kafka.Message{Value: body})
}

func (p *KafkaProducer) PublishBalancesUpdated(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.balancesWriter.WriteMessages(ctx, kafka.Message{Value: body})
}

type Engine struct {
	prod    EventProducer
	nowFunc func() time.Time
}

func NewEngine(prod EventProducer) *Engine {
	return &Engine{
		prod:    prod,
		nowFunc: time.Now,
	}
}

// ProcessMatched consumes a single matched trade event (orders.matched)
// and emits trades.settled and balances.updated after (placeholder) settlement.
func (e *Engine) ProcessMatched(ctx context.Context, t MatchedTrade) error {
	priceDec, err := decimal.NewFromString(t.Price)
	if err != nil {
		log.Printf("settlement: invalid price %q for trade %s", t.Price, t.TradeID)
		return nil
	}
	sizeDec, err := decimal.NewFromString(t.Size)
	if err != nil {
		log.Printf("settlement: invalid size %q for trade %s", t.Size, t.TradeID)
		return nil
	}

	ts := t.Ts
	if ts == "" {
		ts = e.nowFunc().UTC().Format(time.RFC3339Nano)
	}

	tradesSettledPayload := map[string]interface{}{
		"trade_id":       t.TradeID,
		"market_id":      t.MarketID,
		"maker_order_id": t.MakerOrderID,
		"taker_order_id": t.TakerOrderID,
		"price":          priceDec.String(),
		"size":           sizeDec.String(),
		"maker_side":     t.MakerSide,
		"ts":             ts,
	}
	if err := e.prod.PublishTradesSettled(ctx, tradesSettledPayload); err != nil {
		log.Printf("settlement: failed to publish trades.settled: %v", err)
		return err
	}

	// Placeholder: in a real DEX you'd compute per-user per-asset deltas here.
	balancesPayload := map[string]interface{}{
		"note":      "balances.updated would contain per-user per-asset deltas",
		"trade_id":  t.TradeID,
		"market_id": t.MarketID,
		"ts":        ts,
	}
	if err := e.prod.PublishBalancesUpdated(ctx, balancesPayload); err != nil {
		log.Printf("settlement: failed to publish balances.updated: %v", err)
		return err
	}

	return nil
}
