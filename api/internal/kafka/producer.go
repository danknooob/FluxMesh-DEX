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
)

// Producer publishes events to Kafka.
type Producer struct {
	createdWriter  *kafka.Writer
	cancelledWriter *kafka.Writer
}

// NewProducer creates Kafka writers for order topics.
func NewProducer(brokers []string) *Producer {
	addr := kafka.TCP(brokers[0])
	return &Producer{
		createdWriter: &kafka.Writer{
			Addr: addr, Topic: TopicOrdersCreated,
			Balancer: &kafka.LeastBytes{},
		},
		cancelledWriter: &kafka.Writer{
			Addr: addr, Topic: TopicOrdersCancelled,
			Balancer: &kafka.LeastBytes{},
		},
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

// Close closes all writers.
func (p *Producer) Close() error {
	_ = p.createdWriter.Close()
	return p.cancelledWriter.Close()
}
