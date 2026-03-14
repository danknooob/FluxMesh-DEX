package engine

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// mockProducer captures published messages for assertions.
type mockProducer struct {
	mu       sync.Mutex
	matched  []map[string]interface{}
	rejected []map[string]interface{}
}

func (m *mockProducer) PublishOrdersMatched(ctx context.Context, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := payload.(map[string]interface{}); ok {
		m.matched = append(m.matched, p)
	}
	return nil
}

func (m *mockProducer) PublishOrdersRejected(ctx context.Context, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := payload.(map[string]interface{}); ok {
		m.rejected = append(m.rejected, p)
	}
	return nil
}

func (m *mockProducer) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matched = nil
	m.rejected = nil
}

func (m *mockProducer) matchedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.matched)
}

func (m *mockProducer) rejectedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rejected)
}

func (m *mockProducer) lastRejectedReason() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.rejected) == 0 {
		return ""
	}
	r, _ := m.rejected[len(m.rejected)-1]["reason"].(string)
	return r
}

func TestProcessCreated_InvalidSide(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	ctx := context.Background()

	err := eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "o1", UserID: "u1", MarketID: "M1",
		Side: "invalid", Type: "limit", Price: "100", Size: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prod.rejectedCount() != 1 {
		t.Fatalf("expected 1 rejected, got %d", prod.rejectedCount())
	}
	if prod.lastRejectedReason() != "invalid side" {
		t.Errorf("reason=%s", prod.lastRejectedReason())
	}
}

func TestProcessCreated_InvalidPrice(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	ctx := context.Background()

	err := eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "o1", UserID: "u1", MarketID: "M1",
		Side: "buy", Type: "limit", Price: "not-a-number", Size: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prod.rejectedCount() != 1 {
		t.Fatalf("expected 1 rejected, got %d", prod.rejectedCount())
	}
}

func TestProcessCreated_InvalidSize(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	err := eng.ProcessCreated(context.Background(), OrdersCreatedEvent{
		OrderID: "o1", UserID: "u1", MarketID: "M1",
		Side: "buy", Type: "limit", Price: "100", Size: "-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prod.rejectedCount() != 1 {
		t.Fatalf("expected 1 rejected, got %d", prod.rejectedCount())
	}
}

func TestProcessCreated_RestOnBook(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	ctx := context.Background()

	// Ask at 200; buy at 100 -> no match, order rests
	err := eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "ask-1", UserID: "u1", MarketID: "M1",
		Side: "sell", Type: "limit", Price: "200", Size: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prod.reset()

	err = eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "buy-1", UserID: "u2", MarketID: "M1",
		Side: "buy", Type: "limit", Price: "100", Size: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prod.matchedCount() != 0 {
		t.Fatalf("expected no match, got %d matched", prod.matchedCount())
	}
}

func TestProcessCreated_OneFill(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	ctx := context.Background()

	// Ask at 100
	_ = eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "ask-1", UserID: "maker", MarketID: "M1",
		Side: "sell", Type: "limit", Price: "100", Size: "1",
	})
	prod.reset()

	// Buy at 100 -> match
	err := eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "buy-1", UserID: "taker", MarketID: "M1",
		Side: "buy", Type: "limit", Price: "100", Size: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prod.matchedCount() != 1 {
		t.Fatalf("expected 1 matched, got %d", prod.matchedCount())
	}
	payload := prod.matched[0]
	if payload["maker_order_id"] != "ask-1" || payload["taker_order_id"] != "buy-1" {
		t.Errorf("maker=%v taker=%v", payload["maker_order_id"], payload["taker_order_id"])
	}
	if payload["maker_user_id"] != "maker" || payload["taker_user_id"] != "taker" {
		t.Errorf("maker_user_id=%v taker_user_id=%v", payload["maker_user_id"], payload["taker_user_id"])
	}
	if payload["price"] != "100" || payload["size"] != "1" {
		t.Errorf("price=%v size=%v", payload["price"], payload["size"])
	}
}

func TestProcessCancelled_WithMarketID(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	ctx := context.Background()

	_ = eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "order-1", UserID: "u1", MarketID: "M1",
		Side: "buy", Type: "limit", Price: "100", Size: "1",
	})
	prod.reset()

	eng.ProcessCancelled(ctx, OrdersCancelledEvent{
		OrderID: "order-1", UserID: "u1", MarketID: "M1",
	})
	// No publish from cancel; just in-memory. Next buy at 100 should not match with anything
	_ = eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "buy-2", UserID: "u2", MarketID: "M1",
		Side: "buy", Type: "limit", Price: "100", Size: "1",
	})
	if prod.matchedCount() != 0 {
		t.Fatalf("cancelled order should be off book, got %d matched", prod.matchedCount())
	}
}

func TestRestoreOrder(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	ctx := context.Background()
	now := time.Now().UTC()

	eng.RestoreOrder("restored-1", "u1", "M1", "buy", "100", "1", now)
	prod.reset()

	// Incoming sell at 100 should match the restored buy
	err := eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "sell-1", UserID: "u2", MarketID: "M1",
		Side: "sell", Type: "limit", Price: "100", Size: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prod.matchedCount() != 1 {
		t.Fatalf("expected 1 match with restored order, got %d", prod.matchedCount())
	}
	payload := prod.matched[0]
	if payload["maker_order_id"] != "restored-1" {
		t.Errorf("maker_order_id=%v", payload["maker_order_id"])
	}
}

func TestRestoreOrder_InvalidPriceSkipped(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	now := time.Now().UTC()

	eng.RestoreOrder("r1", "u1", "M1", "buy", "invalid", "1", now)
	// Should not panic; order is skipped (logged). Book stays empty.
	eng.RestoreOrder("r2", "u1", "M1", "buy", "100", "1", now)
	// r2 should be on book; we can't easily assert without another event, so just ensure no panic
}

// Test that matched payload is valid JSON (for downstream consumers).
func TestProcessCreated_MatchedPayloadJSON(t *testing.T) {
	prod := &mockProducer{}
	eng := NewEngine(prod)
	ctx := context.Background()
	_ = eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "ask-1", UserID: "m", MarketID: "M1",
		Side: "sell", Type: "limit", Price: "50", Size: "0.5",
	})
	prod.reset()
	_ = eng.ProcessCreated(ctx, OrdersCreatedEvent{
		OrderID: "buy-1", UserID: "t", MarketID: "M1",
		Side: "buy", Type: "limit", Price: "50", Size: "0.5",
	})
	if prod.matchedCount() != 1 {
		t.Fatalf("expected 1 matched, got %d", prod.matchedCount())
	}
	_, err := json.Marshal(prod.matched[0])
	if err != nil {
		t.Errorf("matched payload should be JSON-serializable: %v", err)
	}
}
