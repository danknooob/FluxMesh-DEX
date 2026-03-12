package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

// Topic names used by the data plane.
const (
	TopicOrdersCreated   = "orders.created"
	TopicOrdersCancelled = "orders.cancelled"
	TopicUsersUpdated    = "users.updated"
	TopicUsersDeleted    = "users.deleted"
)

// Producer publishes events to Kafka.
type Producer struct {
	createdWriter   *kafka.Writer
	cancelledWriter *kafka.Writer
	userUpdWriter   *kafka.Writer
	userDelWriter   *kafka.Writer
}

// NewProducer creates Kafka writers for all topics.
func NewProducer(brokers []string) *Producer {
	addr := kafka.TCP(brokers[0])
	w := func(topic string) *kafka.Writer {
		return &kafka.Writer{Addr: addr, Topic: topic, Balancer: &kafka.LeastBytes{}}
	}
	return &Producer{
		createdWriter:   w(TopicOrdersCreated),
		cancelledWriter: w(TopicOrdersCancelled),
		userUpdWriter:   w(TopicUsersUpdated),
		userDelWriter:   w(TopicUsersDeleted),
	}
}

// PublishOrderCreated serializes payload and sends to orders.created.
func (p *Producer) PublishOrderCreated(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.createdWriter, payload)
}

// PublishOrderCancelled sends to orders.cancelled.
func (p *Producer) PublishOrderCancelled(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.cancelledWriter, payload)
}

func (p *Producer) writeJSON(ctx context.Context, w *kafka.Writer, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, kafka.Message{Value: body})
}

// PublishUserUpdated sends to users.updated.
func (p *Producer) PublishUserUpdated(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.userUpdWriter, payload)
}

// PublishUserDeleted sends to users.deleted.
func (p *Producer) PublishUserDeleted(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.userDelWriter, payload)
}

// Close closes all writers.
func (p *Producer) Close() error {
	_ = p.createdWriter.Close()
	_ = p.cancelledWriter.Close()
	_ = p.userUpdWriter.Close()
	return p.userDelWriter.Close()
}
