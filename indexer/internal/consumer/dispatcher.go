package consumer

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/handler"
	"github.com/segmentio/kafka-go"
)

const (
	maxRetries = 3
	baseDelay  = 300 * time.Millisecond
	maxDelay   = 10 * time.Second
)

// Dispatcher creates one Kafka reader per topic and routes messages
// to the corresponding EventHandler from the registry.
type Dispatcher struct {
	brokers  []string
	groupID  string
	registry handler.Registry
}

func NewDispatcher(brokers []string, groupID string, reg handler.Registry) *Dispatcher {
	return &Dispatcher{brokers: brokers, groupID: groupID, registry: reg}
}

// Run starts a goroutine per registered topic. Blocks until ctx is cancelled.
func (d *Dispatcher) Run(ctx context.Context) {
	readers := make([]*kafka.Reader, 0, len(d.registry))

	for topic, h := range d.registry {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     d.brokers,
			GroupID:     d.groupID,
			Topic:       topic,
			MinBytes:    1,
			MaxBytes:    1e6,
			StartOffset: kafka.FirstOffset,
		})
		readers = append(readers, r)
		go d.consumeLoop(ctx, r, topic, h)
	}

	<-ctx.Done()
	for _, r := range readers {
		_ = r.Close()
	}
}

func (d *Dispatcher) consumeLoop(ctx context.Context, r *kafka.Reader, topic string, h handler.EventHandler) {
	log.Printf("indexer: consuming %s", topic)

	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("indexer: fetch error on %s: %v", topic, err)
			time.Sleep(time.Second)
			continue
		}

		if err := d.handleWithRetry(ctx, h, topic, msg); err != nil {
			log.Printf("indexer: DROPPING %s offset %d after %d retries: %v",
				topic, msg.Offset, maxRetries, err)
		}

		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Printf("indexer: commit error on %s offset %d: %v", topic, msg.Offset, err)
		}
	}
}

func (d *Dispatcher) handleWithRetry(ctx context.Context, h handler.EventHandler, topic string, msg kafka.Message) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = h.Handle(ctx, msg.Value)
		if lastErr == nil {
			return nil
		}
		if attempt == maxRetries {
			break
		}
		delay := backoff(attempt)
		log.Printf("indexer: %s offset %d attempt %d/%d failed, retrying in %v: %v",
			topic, msg.Offset, attempt+1, maxRetries, delay, lastErr)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

func backoff(attempt int) time.Duration {
	d := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	if d > maxDelay {
		d = maxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(baseDelay)))
	return d + jitter
}
