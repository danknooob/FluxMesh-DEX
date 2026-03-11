package engine

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/danknooob/fluxmesh-dex/matching-engine/internal/orderbook"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// OrdersCreatedEvent is the payload we expect from the API on orders.created.
type OrdersCreatedEvent struct {
	OrderID  string  `json:"order_id"`
	UserID   string  `json:"user_id"`
	MarketID string  `json:"market_id"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
	Price    string  `json:"price"`
	Size     string  `json:"size"`
	// Remaining omitted here; matching engine tracks remaining as it matches.
}

// EventProducer publishes orders.matched / orders.rejected.
type EventProducer interface {
	PublishOrdersMatched(ctx context.Context, payload interface{}) error
	PublishOrdersRejected(ctx context.Context, payload interface{}) error
}

// KafkaProducer is a basic EventProducer implementation using kafka-go.
type KafkaProducer struct {
	matchedWriter  *kafka.Writer
	rejectedWriter *kafka.Writer
}

// NewKafkaProducer configures writers for the matched and rejected topics.
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

// Engine coordinates per-market books and consumes orders.created.
type Engine struct {
	mu      sync.RWMutex
	books   map[string]orderbook.OrderBook
	prod    EventProducer
	nowFunc func() time.Time
}

// NewEngine creates a new matching engine.
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
	// Double-check after acquiring write lock.
	if book, ok = e.books[marketID]; ok {
		return book
	}
	book = orderbook.NewPriceTimeOrderBook(marketID)
	e.books[marketID] = book
	return book
}

// ProcessCreated handles a single orders.created event from Kafka.
// ProcessCreated applies a single orders.created event to the in-memory books.
// It returns an error only for unexpected failures; validation failures
// result in orders.rejected events and a nil error so the consumer can safely
// commit the offset.
func (e *Engine) ProcessCreated(ctx context.Context, evt OrdersCreatedEvent) error {
	book := e.getBook(evt.MarketID)

	side := orderbook.Side(evt.Side)
	if side != orderbook.SideBuy && side != orderbook.SideSell {
		log.Printf("engine: invalid side %q for order %s", evt.Side, evt.OrderID)
		if err := e.prod.PublishOrdersRejected(ctx, map[string]interface{}{
			"order_id":  evt.OrderID,
			"user_id":   evt.UserID,
			"market_id": evt.MarketID,
			"reason":    "invalid side",
			"ts":        e.nowFunc().UTC(),
		}); err != nil {
			return err
		}
		return nil
	}

	price, err := parseDecimal(evt.Price)
	if err != nil {
		log.Printf("engine: invalid price %q for order %s", evt.Price, evt.OrderID)
		if err := e.prod.PublishOrdersRejected(ctx, map[string]interface{}{
			"order_id":  evt.OrderID,
			"user_id":   evt.UserID,
			"market_id": evt.MarketID,
			"reason":    "invalid price",
			"ts":        e.nowFunc().UTC(),
		}); err != nil {
			return err
		}
		return nil
	}
	size, err := parseDecimal(evt.Size)
	if err != nil {
		log.Printf("engine: invalid size %q for order %s", evt.Size, evt.OrderID)
		if err := e.prod.PublishOrdersRejected(ctx, map[string]interface{}{
			"order_id":  evt.OrderID,
			"user_id":   evt.UserID,
			"market_id": evt.MarketID,
			"reason":    "invalid size",
			"ts":        e.nowFunc().UTC(),
		}); err != nil {
			return err
		}
		return nil
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
		// Rested on the book; nothing else to emit.
		log.Printf("engine: order %s rested on book for market %s", evt.OrderID, evt.MarketID)
		return nil
	}

	// For now we just emit a simple orders.matched event summarizing the fills.
	for _, f := range fills {
		if f.TradeID == "" {
			f.TradeID = uuid.NewString()
		}
		payload := map[string]interface{}{
			"trade_id":       f.TradeID,
			"market_id":      f.MarketID,
			"maker_order_id": f.MakerOrderID,
			"taker_order_id": f.TakerOrderID,
			"price":          f.Price,
			"size":           f.Size,
			"maker_side":     string(f.MakerSide),
			"ts":             f.Ts,
		}
		if err := e.prod.PublishOrdersMatched(ctx, payload); err != nil {
			log.Printf("engine: failed to publish orders.matched: %v", err)
			return err
		}
	}
	return nil
}

func parseDecimal(s string) (float64, error) {
	// For skeleton purposes, use float64; later replace with decimal lib.
	return strconv.ParseFloat(s, 64)
}

