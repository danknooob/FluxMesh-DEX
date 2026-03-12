package model

// OrderMatchedEvent is the Kafka payload from the matching engine.
// Price and Size are decimal strings (shopspring/decimal on the producer side).
type OrderMatchedEvent struct {
	TradeID        string `json:"trade_id"`
	MarketID       string `json:"market_id"`
	MakerOrderID   string `json:"maker_order_id"`
	TakerOrderID   string `json:"taker_order_id"`
	Price          string `json:"price"`
	Size           string `json:"size"`
	MakerSide      string `json:"maker_side"`
	MakerRemaining string `json:"maker_remaining"`
	TakerRemaining string `json:"taker_remaining"`
	Ts             string `json:"ts"`
}

// OrderRejectedEvent is the Kafka payload for a rejected order.
type OrderRejectedEvent struct {
	OrderID  string `json:"order_id"`
	UserID   string `json:"user_id"`
	MarketID string `json:"market_id"`
	Reason   string `json:"reason"`
	Ts       string `json:"ts"`
}

// TradeSettledEvent is the Kafka payload from the settlement service.
type TradeSettledEvent struct {
	TradeID      string `json:"trade_id"`
	MarketID     string `json:"market_id"`
	MakerOrderID string `json:"maker_order_id"`
	TakerOrderID string `json:"taker_order_id"`
	Price        string `json:"price"`
	Size         string `json:"size"`
	MakerSide    string `json:"maker_side"`
	Ts           string `json:"ts"`
}

// BalanceUpdatedEvent is the Kafka payload from the settlement service.
type BalanceUpdatedEvent struct {
	UserID    string `json:"user_id"`
	Asset     string `json:"asset"`
	Available string `json:"available"`
	Locked    string `json:"locked"`
	TradeID   string `json:"trade_id"`
	MarketID  string `json:"market_id"`
	Ts        string `json:"ts"`
}
