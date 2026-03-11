package orderbook

import (
	"sort"
	"time"
)

// Side represents buy or sell.
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// Order models a simplified in-memory order for matching.
type Order struct {
	ID        string
	UserID    string
	MarketID  string
	Side      Side
	Price     float64
	Size      float64
	Remaining float64
	CreatedAt time.Time
}

// Fill represents a single matched piece between two orders.
type Fill struct {
	TradeID      string
	MarketID     string
	MakerOrderID string
	TakerOrderID string
	Price        float64
	Size         float64
	MakerSide    Side
	Ts           time.Time
}

// OrderBook defines the behavior of a per-market order book.
type OrderBook interface {
	MatchIncoming(incoming *Order) (fills []Fill)
	Add(order *Order)
}

// priceTimeOrderBook is a simple implementation with in-memory slices.
// This is sufficient as a skeleton; production systems usually use
// more advanced data structures per price level.
type priceTimeOrderBook struct {
	marketID string
	bids     []*Order // sorted by price desc, then time asc
	asks     []*Order // sorted by price asc, then time asc
}

// NewPriceTimeOrderBook creates a new empty book for a market.
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

	switch in.Side {
	case SideBuy:
		// Match against best asks (ascending price).
		sort.Slice(b.asks, func(i, j int) bool {
			if b.asks[i].Price == b.asks[j].Price {
				return b.asks[i].CreatedAt.Before(b.asks[j].CreatedAt)
			}
			return b.asks[i].Price < b.asks[j].Price
		})
		for _, ask := range b.asks {
			if in.Remaining <= 0 {
				break
			}
			if in.Price < ask.Price {
				break
			}
			size := min(in.Remaining, ask.Remaining)
			if size <= 0 {
				continue
			}
			in.Remaining -= size
			ask.Remaining -= size
			fills = append(fills, Fill{
				TradeID:      "", // to be filled by caller
				MarketID:     b.marketID,
				MakerOrderID: ask.ID,
				TakerOrderID: in.ID,
				Price:        ask.Price,
				Size:         size,
				MakerSide:    SideSell,
				Ts:           now,
			})
		}
		// Clean fully filled asks.
		var remainingAsks []*Order
		for _, ask := range b.asks {
			if ask.Remaining > 0 {
				remainingAsks = append(remainingAsks, ask)
			}
		}
		b.asks = remainingAsks
	case SideSell:
		// Match against best bids (descending price).
		sort.Slice(b.bids, func(i, j int) bool {
			if b.bids[i].Price == b.bids[j].Price {
				return b.bids[i].CreatedAt.Before(b.bids[j].CreatedAt)
			}
			return b.bids[i].Price > b.bids[j].Price
		})
		for _, bid := range b.bids {
			if in.Remaining <= 0 {
				break
			}
			if in.Price > bid.Price {
				break
			}
			size := min(in.Remaining, bid.Remaining)
			if size <= 0 {
				continue
			}
			in.Remaining -= size
			bid.Remaining -= size
			fills = append(fills, Fill{
				TradeID:      "",
				MarketID:     b.marketID,
				MakerOrderID: bid.ID,
				TakerOrderID: in.ID,
				Price:        bid.Price,
				Size:         size,
				MakerSide:    SideBuy,
				Ts:           now,
			})
		}
		// Clean fully filled bids.
		var remainingBids []*Order
		for _, bid := range b.bids {
			if bid.Remaining > 0 {
				remainingBids = append(remainingBids, bid)
			}
		}
		b.bids = remainingBids
	}

	// If the incoming order still has remaining size, rest it on the book.
	if in.Remaining > 0 {
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

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

