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

// DepthBroadcastConsumer consumes order-lifecycle topics and broadcasts
// depth_updated to all WebSocket clients so the order book UI can refresh in real time.
type DepthBroadcastConsumer struct {
	reader *kafka.Reader
	hub    *hub.Hub
	topic  string
}

// NewDepthBroadcastConsumer creates a consumer that emits depth_updated for the given topic.
func NewDepthBroadcastConsumer(brokers []string, topic string, h *hub.Hub) *DepthBroadcastConsumer {
	return &DepthBroadcastConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "notification-depth-" + strings.ReplaceAll(topic, ".", "-"),
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		hub:   h,
		topic: topic,
	}
}

// Run consumes messages and broadcasts depth_updated with market_id to all clients.
func (c *DepthBroadcastConsumer) Run(ctx context.Context) {
	defer func() { _ = c.reader.Close() }()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("notification[depth/%s]: read error: %v", c.topic, err)
			time.Sleep(time.Second)
			continue
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(m.Value, &payload); err != nil {
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		marketID, _ := payload["market_id"].(string)
		if marketID == "" {
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		msg := map[string]string{"type": "depth_updated", "market_id": marketID}
		data, _ := json.Marshal(msg)
		c.hub.BroadcastAll(data)

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("notification[depth/%s]: commit error: %v", c.topic, err)
		}
	}
}
