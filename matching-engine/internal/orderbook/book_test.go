package orderbook

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestNewPriceTimeOrderBook(t *testing.T) {
	book := NewPriceTimeOrderBook("BTC-USDC")
	if book == nil {
		t.Fatal("NewPriceTimeOrderBook returned nil")
	}
}

func TestAddAndCancel_Buy(t *testing.T) {
	book := NewPriceTimeOrderBook("M1")
	now := time.Now().UTC()
	o := &Order{
		ID:        "order-1",
		UserID:    "user-1",
		MarketID:  "M1",
		Side:      SideBuy,
		Price:     decimal.RequireFromString("100"),
		Size:      decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"),
		CreatedAt: now,
	}
	book.Add(o)

	pt := book.(*priceTimeOrderBook)
	if len(pt.bids) != 1 || pt.bids[0].ID != "order-1" {
		t.Fatalf("expected one bid, got %d", len(pt.bids))
	}

	ok := book.Cancel("order-1")
	if !ok {
		t.Fatal("Cancel should have removed order-1")
	}
	if len(pt.bids) != 0 {
		t.Fatalf("expected zero bids after cancel, got %d", len(pt.bids))
	}

	ok = book.Cancel("nonexistent")
	if ok {
		t.Fatal("Cancel nonexistent should return false")
	}
}

func TestAddAndCancel_Sell(t *testing.T) {
	book := NewPriceTimeOrderBook("M1")
	now := time.Now().UTC()
	o := &Order{
		ID: "order-2", UserID: "u2", MarketID: "M1", Side: SideSell,
		Price: decimal.RequireFromString("200"), Size: decimal.RequireFromString("2"),
		Remaining: decimal.RequireFromString("2"), CreatedAt: now,
	}
	book.Add(o)
	pt := book.(*priceTimeOrderBook)
	if len(pt.asks) != 1 {
		t.Fatalf("expected one ask, got %d", len(pt.asks))
	}
	ok := book.Cancel("order-2")
	if !ok {
		t.Fatal("Cancel sell should succeed")
	}
	if len(pt.asks) != 0 {
		t.Fatalf("expected zero asks after cancel, got %d", len(pt.asks))
	}
}

func TestMatchIncoming_BuyNoMatch(t *testing.T) {
	book := NewPriceTimeOrderBook("M1")
	now := time.Now().UTC()
	// Single ask at 200
	book.Add(&Order{
		ID: "ask-1", UserID: "maker", MarketID: "M1", Side: SideSell,
		Price: decimal.RequireFromString("200"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	})
	// Buy at 100: no match
	in := &Order{
		ID: "buy-1", UserID: "taker", MarketID: "M1", Side: SideBuy,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	}
	fills := book.MatchIncoming(in)
	if len(fills) != 0 {
		t.Fatalf("expected no fills, got %d", len(fills))
	}
	pt := book.(*priceTimeOrderBook)
	if len(pt.bids) != 1 || pt.bids[0].ID != "buy-1" {
		t.Fatalf("buy order should rest on book: bids=%d", len(pt.bids))
	}
}

func TestMatchIncoming_BuyFullFill(t *testing.T) {
	book := NewPriceTimeOrderBook("M1")
	now := time.Now().UTC()
	book.Add(&Order{
		ID: "ask-1", UserID: "maker", MarketID: "M1", Side: SideSell,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	})
	in := &Order{
		ID: "buy-1", UserID: "taker", MarketID: "M1", Side: SideBuy,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	}
	fills := book.MatchIncoming(in)
	if len(fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(fills))
	}
	f := fills[0]
	if f.MakerOrderID != "ask-1" || f.TakerOrderID != "buy-1" {
		t.Errorf("maker=%s taker=%s", f.MakerOrderID, f.TakerOrderID)
	}
	if !f.Size.Equal(decimal.RequireFromString("1")) {
		t.Errorf("fill size=%s", f.Size.String())
	}
	if !f.MakerRemaining.IsZero() || !f.TakerRemaining.IsZero() {
		t.Errorf("both should be fully filled: maker_rem=%s taker_rem=%s", f.MakerRemaining.String(), f.TakerRemaining.String())
	}
	pt := book.(*priceTimeOrderBook)
	if len(pt.asks) != 0 {
		t.Fatalf("maker ask should be removed, asks=%d", len(pt.asks))
	}
	if in.Remaining.GreaterThan(decimal.Zero) {
		t.Errorf("incoming remaining should be 0, got %s", in.Remaining.String())
	}
}

func TestMatchIncoming_BuyPartialFill(t *testing.T) {
	book := NewPriceTimeOrderBook("M1")
	now := time.Now().UTC()
	book.Add(&Order{
		ID: "ask-1", UserID: "maker", MarketID: "M1", Side: SideSell,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("2"),
		Remaining: decimal.RequireFromString("2"), CreatedAt: now,
	})
	in := &Order{
		ID: "buy-1", UserID: "taker", MarketID: "M1", Side: SideBuy,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	}
	fills := book.MatchIncoming(in)
	if len(fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(fills))
	}
	if !fills[0].Size.Equal(decimal.RequireFromString("1")) {
		t.Errorf("fill size=%s", fills[0].Size.String())
	}
	// Maker remaining 1, taker 0
	if !fills[0].MakerRemaining.Equal(decimal.RequireFromString("1")) {
		t.Errorf("maker remaining=%s", fills[0].MakerRemaining.String())
	}
	if !fills[0].TakerRemaining.IsZero() {
		t.Errorf("taker remaining=%s", fills[0].TakerRemaining.String())
	}
	pt := book.(*priceTimeOrderBook)
	if len(pt.asks) != 1 || !pt.asks[0].Remaining.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("one ask with remaining 1: asks=%d", len(pt.asks))
	}
}

func TestMatchIncoming_SellFullFill(t *testing.T) {
	book := NewPriceTimeOrderBook("M1")
	now := time.Now().UTC()
	book.Add(&Order{
		ID: "bid-1", UserID: "maker", MarketID: "M1", Side: SideBuy,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	})
	in := &Order{
		ID: "sell-1", UserID: "taker", MarketID: "M1", Side: SideSell,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	}
	fills := book.MatchIncoming(in)
	if len(fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(fills))
	}
	if fills[0].MakerOrderID != "bid-1" || fills[0].TakerOrderID != "sell-1" {
		t.Errorf("maker=%s taker=%s", fills[0].MakerOrderID, fills[0].TakerOrderID)
	}
	pt := book.(*priceTimeOrderBook)
	if len(pt.bids) != 0 {
		t.Fatalf("maker bid should be removed, bids=%d", len(pt.bids))
	}
}

func TestMatchIncoming_TwoMakers(t *testing.T) {
	book := NewPriceTimeOrderBook("M1")
	now := time.Now().UTC()
	book.Add(&Order{
		ID: "ask-1", UserID: "m1", MarketID: "M1", Side: SideSell,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	})
	now = now.Add(time.Millisecond)
	book.Add(&Order{
		ID: "ask-2", UserID: "m2", MarketID: "M1", Side: SideSell,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("1"),
		Remaining: decimal.RequireFromString("1"), CreatedAt: now,
	})
	in := &Order{
		ID: "buy-1", UserID: "taker", MarketID: "M1", Side: SideBuy,
		Price: decimal.RequireFromString("100"), Size: decimal.RequireFromString("2"),
		Remaining: decimal.RequireFromString("2"), CreatedAt: now.Add(time.Millisecond),
	}
	fills := book.MatchIncoming(in)
	if len(fills) != 2 {
		t.Fatalf("expected 2 fills, got %d", len(fills))
	}
	if fills[0].MakerOrderID != "ask-1" || fills[1].MakerOrderID != "ask-2" {
		t.Errorf("price-time priority: first fill maker should be ask-1, got %s and %s", fills[0].MakerOrderID, fills[1].MakerOrderID)
	}
	pt := book.(*priceTimeOrderBook)
	if len(pt.asks) != 0 {
		t.Fatalf("both asks filled, asks=%d", len(pt.asks))
	}
	if in.Remaining.GreaterThan(decimal.Zero) {
		t.Errorf("taker should be fully filled, remaining=%s", in.Remaining.String())
	}
}

func TestPruneFilledOrders(t *testing.T) {
	o1 := &Order{Remaining: decimal.Zero}
	o2 := &Order{Remaining: decimal.RequireFromString("1")}
	out := pruneFilledOrders([]*Order{o1, o2})
	if len(out) != 1 || out[0] != o2 {
		t.Fatalf("expected one order kept, got %d", len(out))
	}
}
