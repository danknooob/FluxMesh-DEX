package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TopicOrdersCreated   = "orders.created"
	TopicOrdersCancelled = "orders.cancelled"
	TopicUsersUpdated    = "users.updated"
	TopicUsersDeleted    = "users.deleted"

	maxRetries   = 3
	baseDelay    = 200 * time.Millisecond
	maxDelay     = 5 * time.Second
)

// Producer publishes events to Kafka with automatic retry on transient errors.
type Producer struct {
	createdWriter   *kafka.Writer
	cancelledWriter *kafka.Writer
	userUpdWriter   *kafka.Writer
	userDelWriter   *kafka.Writer
}

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

func (p *Producer) PublishOrderCreated(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.createdWriter, payload)
}

func (p *Producer) PublishOrderCancelled(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.cancelledWriter, payload)
}

func (p *Producer) PublishUserUpdated(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.userUpdWriter, payload)
}

func (p *Producer) PublishUserDeleted(ctx context.Context, payload interface{}) error {
	return p.writeJSON(ctx, p.userDelWriter, payload)
}

func (p *Producer) writeJSON(ctx context.Context, w *kafka.Writer, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := kafka.Message{Value: body}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = w.WriteMessages(ctx, msg)
		if err == nil {
			return nil
		}
		if !isTransient(err) {
			return err
		}
		if attempt == maxRetries {
			break
		}
		delay := backoff(attempt)
		log.Printf("kafka-producer: transient error on %s (attempt %d/%d), retrying in %v: %v",
			w.Topic, attempt+1, maxRetries, delay, err)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

func isTransient(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return true // kafka-go connection/broker errors are generally transient
}

func backoff(attempt int) time.Duration {
	d := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	if d > maxDelay {
		d = maxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(baseDelay)))
	return d + jitter
}

func (p *Producer) Close() error {
	_ = p.createdWriter.Close()
	_ = p.cancelledWriter.Close()
	_ = p.userUpdWriter.Close()
	return p.userDelWriter.Close()
}
