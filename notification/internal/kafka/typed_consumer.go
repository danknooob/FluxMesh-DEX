package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/danknooob/fluxmesh-dex/notification/internal/hub"
	"github.com/segmentio/kafka-go"
)

// TypedConsumer is like NotificationConsumer but injects a "type" field
// into the forwarded JSON so the frontend can distinguish event kinds.
type TypedConsumer struct {
	reader    *kafka.Reader
	hub       *hub.Hub
	eventType string
}

func NewTypedConsumer(brokers []string, topic, eventType string, h *hub.Hub) *TypedConsumer {
	return &TypedConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "notification-" + strings.ReplaceAll(topic, ".", "-"),
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		hub:       h,
		eventType: eventType,
	}
}

func (c *TypedConsumer) Run(ctx context.Context) {
	defer c.reader.Close()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("notification[%s]: read error: %v", c.eventType, err)
			time.Sleep(time.Second)
			continue
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(m.Value, &payload); err != nil {
			log.Printf("notification[%s]: unmarshal error: %v", c.eventType, err)
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		uid, _ := payload["user_id"].(string)
		if uid == "" {
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		payload["type"] = c.eventType
		wrapped, _ := json.Marshal(payload)
		c.hub.Broadcast(uid, wrapped)

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("notification[%s]: commit error: %v", c.eventType, err)
		}
	}
}

// MatchedConsumer handles orders.matched events which contain two users
// (maker + taker). It sends a typed notification to each.
type MatchedConsumer struct {
	reader *kafka.Reader
	hub    *hub.Hub
}

func NewMatchedConsumer(brokers []string, h *hub.Hub) *MatchedConsumer {
	return &MatchedConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    "orders.matched",
			GroupID:  "notification-orders-matched",
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		hub: h,
	}
}

func (c *MatchedConsumer) Run(ctx context.Context) {
	defer c.reader.Close()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("notification[matched]: read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(m.Value, &payload); err != nil {
			log.Printf("notification[matched]: unmarshal error: %v", err)
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		makerUID, _ := payload["maker_user_id"].(string)
		takerUID, _ := payload["taker_user_id"].(string)

		base := map[string]interface{}{
			"type":      "order_filled",
			"trade_id":  payload["trade_id"],
			"market_id": payload["market_id"],
			"price":     payload["price"],
			"size":      payload["size"],
		}

		if makerUID != "" {
			msg := copyMap(base)
			msg["order_id"] = payload["maker_order_id"]
			msg["remaining"] = payload["maker_remaining"]
			msg["role"] = "maker"
			data, _ := json.Marshal(msg)
			c.hub.Broadcast(makerUID, data)
		}

		if takerUID != "" {
			msg := copyMap(base)
			msg["order_id"] = payload["taker_order_id"]
			msg["remaining"] = payload["taker_remaining"]
			msg["role"] = "taker"
			data, _ := json.Marshal(msg)
			c.hub.Broadcast(takerUID, data)
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("notification[matched]: commit error: %v", err)
		}
	}
}

func copyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
