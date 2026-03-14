package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/danknooob/fluxmesh-dex/api/internal/metrics"
	"github.com/segmentio/kafka-go"
	"github.com/sony/gobreaker"
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

// Circuit breaker: open after 5 consecutive failures, try again after 30s.
const (
	cbTimeout     = 30 * time.Second
	cbReadyToTrip = 5
)

// Producer publishes events to Kafka with automatic retry on transient errors
// and a circuit breaker so sustained broker failures fail fast instead of hammering Kafka.
type Producer struct {
	createdWriter   *kafka.Writer
	cancelledWriter *kafka.Writer
	userUpdWriter   *kafka.Writer
	userDelWriter   *kafka.Writer
	cb              *gobreaker.CircuitBreaker
}

func NewProducer(brokers []string) *Producer {
	if len(brokers) == 0 {
		panic("kafka producer: at least one broker required")
	}
	addr := kafka.TCP(brokers[0])
	w := func(topic string) *kafka.Writer {
		return &kafka.Writer{Addr: addr, Topic: topic, Balancer: &kafka.LeastBytes{}}
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    "kafka-producer",
		Timeout: cbTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cbReadyToTrip
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Info("kafka-producer circuit breaker state change", "name", name, "from", from.String(), "to", to.String())
		},
		// Don't count context cancellation as failure so client timeouts don't open the circuit.
		IsSuccessful: func(err error) bool {
			return err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
		},
	})
	return &Producer{
		createdWriter:   w(TopicOrdersCreated),
		cancelledWriter: w(TopicOrdersCancelled),
		userUpdWriter:   w(TopicUsersUpdated),
		userDelWriter:   w(TopicUsersDeleted),
		cb:              cb,
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
		_, err = p.cb.Execute(func() (interface{}, error) {
			return nil, w.WriteMessages(ctx, msg)
		})
		if err != nil {
			// Circuit open or half-open (too many probes): fail fast, do not retry
			if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
				metrics.ObserveKafkaPublish(w.Topic, err)
				return err
			}
			if !isTransient(err) {
				metrics.ObserveKafkaPublish(w.Topic, err)
				return err
			}
			if attempt == maxRetries {
				break
			}
			delay := backoff(attempt)
			slog.Warn("kafka-producer transient error, retrying", "topic", w.Topic, "attempt", attempt+1, "max_retries", maxRetries, "delay", delay, "error", err)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				metrics.ObserveKafkaPublish(w.Topic, ctx.Err())
				return ctx.Err()
			}
			continue
		}
		metrics.ObserveKafkaPublish(w.Topic, nil)
		return nil
	}
	metrics.ObserveKafkaPublish(w.Topic, err)
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
	var firstErr error
	for _, w := range []*kafka.Writer{p.createdWriter, p.cancelledWriter, p.userUpdWriter, p.userDelWriter} {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
