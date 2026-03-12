package consumer

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/danknooob/fluxmesh-dex/eventlog/internal/store"
	"github.com/segmentio/kafka-go"
)

const (
	mongoMaxRetries = 4
	mongoBaseDelay  = 300 * time.Millisecond
	mongoMaxDelay   = 10 * time.Second
)

// TopicConsumer reads messages from a set of Kafka topics and persists them
// to MongoDB via the EventStore with retry-on-failure semantics.
type TopicConsumer struct {
	readers []*kafka.Reader
	store   store.EventStore
}

func New(brokers []string, groupID string, topics []string, es store.EventStore) *TopicConsumer {
	readers := make([]*kafka.Reader, len(topics))
	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       1,
			MaxBytes:       1e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		})
	}
	return &TopicConsumer{readers: readers, store: es}
}

func (tc *TopicConsumer) Run(ctx context.Context) {
	for _, r := range tc.readers {
		go tc.consumeLoop(ctx, r)
	}
	<-ctx.Done()
	for _, r := range tc.readers {
		r.Close()
	}
}

func (tc *TopicConsumer) consumeLoop(ctx context.Context, r *kafka.Reader) {
	topic := r.Config().Topic
	log.Printf("eventlog: consuming topic %s", topic)

	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("eventlog: fetch error on %s: %v", topic, err)
			time.Sleep(time.Second)
			continue
		}

		payload := store.ParsePayload(msg.Value)
		doc := &store.EventDocument{
			Topic:     msg.Topic,
			Title:     store.TitleForEvent(msg.Topic, payload),
			Key:       string(msg.Key),
			Payload:   payload,
			Offset:    msg.Offset,
			Partition: msg.Partition,
			Timestamp: msg.Time,
		}

		if err := tc.saveWithRetry(ctx, doc, topic, msg.Offset); err != nil {
			log.Printf("eventlog: DROPPING event on %s offset %d after %d retries: %v",
				topic, msg.Offset, mongoMaxRetries, err)
		}

		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Printf("eventlog: commit error on %s offset %d: %v", topic, msg.Offset, err)
		}
	}
}

func (tc *TopicConsumer) saveWithRetry(ctx context.Context, doc *store.EventDocument, topic string, offset int64) error {
	var lastErr error
	for attempt := 0; attempt <= mongoMaxRetries; attempt++ {
		lastErr = tc.store.Save(ctx, doc)
		if lastErr == nil {
			return nil
		}
		if attempt == mongoMaxRetries {
			break
		}
		delay := backoff(attempt)
		log.Printf("eventlog: mongo save error on %s offset %d (attempt %d/%d), retrying in %v: %v",
			topic, offset, attempt+1, mongoMaxRetries, delay, lastErr)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

func backoff(attempt int) time.Duration {
	d := time.Duration(float64(mongoBaseDelay) * math.Pow(2, float64(attempt)))
	if d > mongoMaxDelay {
		d = mongoMaxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(mongoBaseDelay)))
	return d + jitter
}
