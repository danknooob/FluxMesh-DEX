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

// NotificationConsumer subscribes to domain topics and pushes messages to users via the hub.
type NotificationConsumer struct {
	reader *kafka.Reader
	hub    *hub.Hub
}

// NewNotificationConsumer creates a consumer on the given topic.
func NewNotificationConsumer(brokers []string, topic string, h *hub.Hub) *NotificationConsumer {
	return &NotificationConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "notification-" + strings.ReplaceAll(topic, ".", "-"),
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		hub: h,
	}
}

// Run starts consuming messages until the context is cancelled.
// It expects each payload to include a user_id field, either at the top level
// or inside a "user_id" key, and forwards the raw JSON to that user's clients.
func (c *NotificationConsumer) Run(ctx context.Context) {
	defer func() { _ = c.reader.Close() }()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("notification: shutting down consumer")
				return
			}
			log.Printf("notification: read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(m.Value, &payload); err != nil {
			log.Printf("notification: failed to unmarshal message: %v", err)
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		uid, _ := payload["user_id"].(string)
		if uid == "" {
			// If no user_id, we currently drop the message.
			log.Printf("notification: message missing user_id, dropping")
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		// Broadcast raw JSON to this user.
		c.hub.Broadcast(uid, m.Value)

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("notification: commit failed: %v", err)
			time.Sleep(time.Second)
		}
	}
}

