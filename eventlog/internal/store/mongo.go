package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EventDocument is the shape of every event persisted in MongoDB.
type EventDocument struct {
	Topic     string    `bson:"topic"`
	Title     string    `bson:"title"`
	Key       string    `bson:"key,omitempty"`
	Payload   bson.M    `bson:"payload"`
	Offset    int64     `bson:"offset"`
	Partition int       `bson:"partition"`
	Timestamp time.Time `bson:"timestamp"`
	StoredAt  time.Time `bson:"stored_at"`
}

// EventStore persists Kafka events into MongoDB.
type EventStore interface {
	Save(ctx context.Context, doc *EventDocument) error
	Close(ctx context.Context) error
}

type mongoStore struct {
	client *mongo.Client
	coll   *mongo.Collection
}

// NewMongoStore connects to MongoDB and returns an EventStore.
// Each Kafka event is stored in the "events" collection inside the given database.
func NewMongoStore(ctx context.Context, uri, database string) (EventStore, error) {
	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	coll := client.Database(database).Collection("events")

	indexModels := []mongo.IndexModel{
		{Keys: bson.D{bson.E{Key: "topic", Value: 1}, bson.E{Key: "timestamp", Value: -1}}},
		{Keys: bson.D{bson.E{Key: "title", Value: 1}}},
		{Keys: bson.D{bson.E{Key: "stored_at", Value: 1}}},
	}
	_ = coll.Indexes().CreateMany(ctx, indexModels)

	return &mongoStore{client: client, coll: coll}, nil
}

func (s *mongoStore) Save(ctx context.Context, doc *EventDocument) error {
	doc.StoredAt = time.Now().UTC()
	_, err := s.coll.InsertOne(ctx, doc)
	return err
}

func (s *mongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// ParsePayload attempts to unmarshal a JSON byte slice into a bson.M.
// Falls back to storing the raw string if the payload is not valid JSON.
func ParsePayload(raw []byte) bson.M {
	var m bson.M
	if err := json.Unmarshal(raw, &m); err != nil {
		return bson.M{"raw": string(raw)}
	}
	return m
}

// TitleForEvent generates a human-readable title for a given topic and payload.
func TitleForEvent(topic string, payload bson.M) string {
	str := func(key string) string {
		if v, ok := payload[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	switch topic {
	case "orders.created":
		side := str("side")
		market := str("market_id")
		size := str("size")
		price := str("price")
		if market != "" {
			return fmt.Sprintf("New %s order: %s %s @ %s", market, side, size, price)
		}
		return "New order created"

	case "orders.cancelled":
		orderID := str("order_id")
		if orderID == "" {
			orderID = str("id")
		}
		return fmt.Sprintf("Order cancelled: %s", orderID)

	case "orders.matched":
		market := str("market_id")
		fills := str("fill_count")
		if fills == "" {
			fills = "N/A"
		}
		return fmt.Sprintf("Order matched on %s (%s fills)", market, fills)

	case "orders.rejected":
		reason := str("reason")
		orderID := str("order_id")
		if orderID == "" {
			orderID = str("id")
		}
		if reason != "" {
			return fmt.Sprintf("Order rejected: %s — %s", orderID, reason)
		}
		return fmt.Sprintf("Order rejected: %s", orderID)

	case "trades.settled":
		market := str("market_id")
		count := str("trade_count")
		if count == "" {
			count = "batch"
		}
		return fmt.Sprintf("Trades settled on %s (%s trades)", market, count)

	case "balances.updated":
		userID := str("user_id")
		asset := str("asset")
		if userID != "" && asset != "" {
			return fmt.Sprintf("Balance updated: user %s, asset %s", userID, asset)
		}
		return "Balance updated"

	case "notifications.user":
		nType := str("type")
		userID := str("user_id")
		if nType != "" {
			return fmt.Sprintf("Notification [%s] for user %s", nType, userID)
		}
		return "User notification"

	case "control.config":
		action := str("action")
		target := str("market_id")
		if target == "" {
			target = str("key")
		}
		if action != "" {
			return fmt.Sprintf("Config change: %s %s", action, target)
		}
		return "Configuration updated"

	case "control.health":
		service := str("service")
		status := str("status")
		if service != "" {
			return fmt.Sprintf("Health heartbeat: %s is %s", service, status)
		}
		return "Service health heartbeat"

	case "control.audit":
		action := str("action")
		admin := str("admin_id")
		if admin == "" {
			admin = str("user_id")
		}
		if action != "" {
			return fmt.Sprintf("Audit: %s by %s", action, admin)
		}
		return "Audit event"

	case "control.commands":
		command := str("command")
		target := str("target")
		if command != "" {
			return fmt.Sprintf("Command: %s → %s", command, target)
		}
		return "Control command issued"

	case "users.updated":
		userID := str("user_id")
		action := str("action")
		name := str("name")
		newEmail := str("new_email")
		if newEmail != "" {
			return fmt.Sprintf("Profile updated: user %s changed email to %s", userID, newEmail)
		}
		if name != "" {
			return fmt.Sprintf("Profile updated: user %s set name to %s", userID, name)
		}
		if action != "" {
			return fmt.Sprintf("Profile updated: user %s (%s)", userID, action)
		}
		return fmt.Sprintf("Profile updated: user %s", userID)

	case "users.deleted":
		userID := str("user_id")
		email := str("email")
		if email != "" {
			return fmt.Sprintf("Account deleted: %s (%s)", email, userID)
		}
		return fmt.Sprintf("Account deleted: user %s", userID)

	default:
		return fmt.Sprintf("Event on %s", topic)
	}
}
