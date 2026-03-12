package engine

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/danknooob/fluxmesh-dex/matching-engine/internal/orderbook"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
)

type OrdersCreatedEvent struct {
	OrderID  string `json:"order_id"`
	UserID   string `json:"user_id"`
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Type     string `json:"type"`
	Price    string `json:"price"`
	Size     string `json:"size"`
}

type OrdersCancelledEvent struct {
	OrderID  string `json:"order_id"`
	UserID   string `json:"user_id"`
	MarketID string `json:"market_id"`
}

type EventProducer interface {
	PublishOrdersMatched(ctx context.Context, payload interface{}) error
	PublishOrdersRejected(ctx context.Context, payload interface{}) error
}

type KafkaProducer struct {
	matchedWriter  *kafka.Writer
	rejectedWriter *kafka.Writer
}

func NewKafkaProducer(brokers []string, matchedTopic, rejectedTopic string) *KafkaProducer {
	addr := kafka.TCP(brokers...)
	return &KafkaProducer{
		matchedWriter: &kafka.Writer{
			Addr:     addr,
			Topic:    matchedTopic,
			Balancer: &kafka.LeastBytes{},
		},
		rejectedWriter: &kafka.Writer{
			Addr:     addr,
			Topic:    rejectedTopic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *KafkaProducer) PublishOrdersMatched(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.matchedWriter.WriteMessages(ctx, kafka.Message{Value: body})
}

func (p *KafkaProducer) PublishOrdersRejected(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.rejectedWriter.WriteMessages(ctx, kafka.Message{Value: body})
}

type Engine struct {
	mu      sync.RWMutex
	books   map[string]orderbook.OrderBook
	prod    EventProducer
	nowFunc func() time.Time
}

func NewEngine(prod EventProducer) *Engine {
	return &Engine{
		books:   make(map[string]orderbook.OrderBook),
		prod:    prod,
		nowFunc: time.Now,
	}
}

func (e *Engine) getBook(marketID string) orderbook.OrderBook {
	e.mu.RLock()
	book, ok := e.books[marketID]
	e.mu.RUnlock()
	if ok {
		return book
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if book, ok = e.books[marketID]; ok {
		return book
	}
	book = orderbook.NewPriceTimeOrderBook(marketID)
	e.books[marketID] = book
	return book
}

func (e *Engine) ProcessCreated(ctx context.Context, evt OrdersCreatedEvent) error {
	book := e.getBook(evt.MarketID)

	side := orderbook.Side(evt.Side)
	if side != orderbook.SideBuy && side != orderbook.SideSell {
		log.Printf("engine: invalid side %q for order %s", evt.Side, evt.OrderID)
		return e.reject(ctx, evt, "invalid side")
	}

	price, err := decimal.NewFromString(evt.Price)
	if err != nil {
		log.Printf("engine: invalid price %q for order %s", evt.Price, evt.OrderID)
		return e.reject(ctx, evt, "invalid price")
	}
	size, err := decimal.NewFromString(evt.Size)
	if err != nil {
		log.Printf("engine: invalid size %q for order %s", evt.Size, evt.OrderID)
		return e.reject(ctx, evt, "invalid size")
	}
	if price.LessThanOrEqual(decimal.Zero) {
		log.Printf("engine: non-positive price for order %s", evt.OrderID)
		return e.reject(ctx, evt, "price must be positive")
	}
	if size.LessThanOrEqual(decimal.Zero) {
		log.Printf("engine: non-positive size for order %s", evt.OrderID)
		return e.reject(ctx, evt, "size must be positive")
	}

	incoming := &orderbook.Order{
		ID:        evt.OrderID,
		UserID:    evt.UserID,
		MarketID:  evt.MarketID,
		Side:      side,
		Price:     price,
		Size:      size,
		Remaining: size,
		CreatedAt: e.nowFunc().UTC(),
	}

	fills := book.MatchIncoming(incoming)
	if len(fills) == 0 {
		log.Printf("engine: order %s rested on book for market %s", evt.OrderID, evt.MarketID)
		return nil
	}

	for _, f := range fills {
		if f.TradeID == "" {
			f.TradeID = uuid.NewString()
		}
		payload := map[string]interface{}{
			"trade_id":         f.TradeID,
			"market_id":        f.MarketID,
			"maker_order_id":   f.MakerOrderID,
			"taker_order_id":   f.TakerOrderID,
			"price":            f.Price.String(),
			"size":             f.Size.String(),
			"maker_side":       string(f.MakerSide),
			"maker_remaining":  f.MakerRemaining.String(),
			"taker_remaining":  f.TakerRemaining.String(),
			"ts":               f.Ts,
		}
		if err := e.prod.PublishOrdersMatched(ctx, payload); err != nil {
			log.Printf("engine: failed to publish orders.matched: %v", err)
			return err
		}
	}
	return nil
}

// ProcessCancelled removes a resting order from the in-memory book.
// If the event carries a market_id, we target that book directly;
// otherwise we fall back to scanning all books.
func (e *Engine) ProcessCancelled(_ context.Context, evt OrdersCancelledEvent) {
	if evt.MarketID != "" {
		e.mu.RLock()
		book, ok := e.books[evt.MarketID]
		e.mu.RUnlock()
		if ok && book.Cancel(evt.OrderID) {
			log.Printf("engine: cancelled order %s on market %s (user %s)",
				evt.OrderID, evt.MarketID, evt.UserID)
			return
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, book := range e.books {
		if book.Cancel(evt.OrderID) {
			log.Printf("engine: cancelled order %s (user %s)", evt.OrderID, evt.UserID)
			return
		}
	}
	log.Printf("engine: cancel order %s not found on any book (already filled or never rested)", evt.OrderID)
}

func (e *Engine) reject(ctx context.Context, evt OrdersCreatedEvent, reason string) error {
	return e.prod.PublishOrdersRejected(ctx, map[string]interface{}{
		"order_id":  evt.OrderID,
		"user_id":   evt.UserID,
		"market_id": evt.MarketID,
		"reason":    reason,
		"ts":        e.nowFunc().UTC(),
	})
}
