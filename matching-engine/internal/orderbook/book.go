package orderbook

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Order struct {
	ID        string
	UserID    string
	MarketID  string
	Side      Side
	Price     decimal.Decimal
	Size      decimal.Decimal
	Remaining decimal.Decimal
	CreatedAt time.Time
}

type Fill struct {
	TradeID        string
	MarketID       string
	MakerOrderID   string
	TakerOrderID   string
	Price          decimal.Decimal
	Size           decimal.Decimal
	MakerSide      Side
	MakerRemaining decimal.Decimal
	TakerRemaining decimal.Decimal
	Ts             time.Time
}

type OrderBook interface {
	MatchIncoming(incoming *Order) (fills []Fill)
	Add(order *Order)
	Cancel(orderID string) bool
}

type priceTimeOrderBook struct {
	marketID string
	bids     []*Order
	asks     []*Order
}

func NewPriceTimeOrderBook(marketID string) OrderBook {
	return &priceTimeOrderBook{
		marketID: marketID,
		bids:     []*Order{},
		asks:     []*Order{},
	}
}

func (b *priceTimeOrderBook) MatchIncoming(in *Order) []Fill {
	var fills []Fill
	now := time.Now().UTC()
	zero := decimal.Zero

	switch in.Side {
	case SideBuy:
		sort.Slice(b.asks, func(i, j int) bool {
			cmp := b.asks[i].Price.Cmp(b.asks[j].Price)
			if cmp == 0 {
				return b.asks[i].CreatedAt.Before(b.asks[j].CreatedAt)
			}
			return cmp < 0
		})
		for _, ask := range b.asks {
			if in.Remaining.LessThanOrEqual(zero) {
				break
			}
			if in.Price.LessThan(ask.Price) {
				break
			}
			fillSize := decimal.Min(in.Remaining, ask.Remaining)
			if fillSize.LessThanOrEqual(zero) {
				continue
			}
			in.Remaining = in.Remaining.Sub(fillSize)
			ask.Remaining = ask.Remaining.Sub(fillSize)
			fills = append(fills, Fill{
				MarketID:       b.marketID,
				MakerOrderID:   ask.ID,
				TakerOrderID:   in.ID,
				Price:          ask.Price,
				Size:           fillSize,
				MakerSide:      SideSell,
				MakerRemaining: ask.Remaining,
				TakerRemaining: in.Remaining,
				Ts:             now,
			})
		}
		b.asks = pruneFilledOrders(b.asks)

	case SideSell:
		sort.Slice(b.bids, func(i, j int) bool {
			cmp := b.bids[i].Price.Cmp(b.bids[j].Price)
			if cmp == 0 {
				return b.bids[i].CreatedAt.Before(b.bids[j].CreatedAt)
			}
			return cmp > 0
		})
		for _, bid := range b.bids {
			if in.Remaining.LessThanOrEqual(zero) {
				break
			}
			if in.Price.GreaterThan(bid.Price) {
				break
			}
			fillSize := decimal.Min(in.Remaining, bid.Remaining)
			if fillSize.LessThanOrEqual(zero) {
				continue
			}
			in.Remaining = in.Remaining.Sub(fillSize)
			bid.Remaining = bid.Remaining.Sub(fillSize)
			fills = append(fills, Fill{
				MarketID:       b.marketID,
				MakerOrderID:   bid.ID,
				TakerOrderID:   in.ID,
				Price:          bid.Price,
				Size:           fillSize,
				MakerSide:      SideBuy,
				MakerRemaining: bid.Remaining,
				TakerRemaining: in.Remaining,
				Ts:             now,
			})
		}
		b.bids = pruneFilledOrders(b.bids)
	}

	if in.Remaining.GreaterThan(zero) {
		b.Add(in)
	}

	return fills
}

func (b *priceTimeOrderBook) Add(order *Order) {
	switch order.Side {
	case SideBuy:
		b.bids = append(b.bids, order)
	case SideSell:
		b.asks = append(b.asks, order)
	}
}

// Cancel removes a resting order from the book by ID.
// Returns true if the order was found and removed.
func (b *priceTimeOrderBook) Cancel(orderID string) bool {
	for i, o := range b.bids {
		if o.ID == orderID {
			b.bids = append(b.bids[:i], b.bids[i+1:]...)
			return true
		}
	}
	for i, o := range b.asks {
		if o.ID == orderID {
			b.asks = append(b.asks[:i], b.asks[i+1:]...)
			return true
		}
	}
	return false
}

func pruneFilledOrders(orders []*Order) []*Order {
	kept := make([]*Order, 0, len(orders))
	for _, o := range orders {
		if o.Remaining.GreaterThan(decimal.Zero) {
			kept = append(kept, o)
		}
	}
	return kept
}
