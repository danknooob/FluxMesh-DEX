package consumer

import (
	"context"
	"log"
	"time"

	"github.com/danknooob/fluxmesh-dex/eventlog/internal/store"
	"github.com/segmentio/kafka-go"
)

// TopicConsumer reads messages from a set of Kafka topics and persists them
// to MongoDB via the EventStore.
type TopicConsumer struct {
	readers []*kafka.Reader
	store   store.EventStore
}

// New creates a TopicConsumer that subscribes to the given topics.
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

// Run starts a goroutine per topic that reads and persists events.
// It blocks until ctx is cancelled.
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

		if err := tc.store.Save(ctx, doc); err != nil {
			log.Printf("eventlog: mongo save error on %s offset %d: %v", topic, msg.Offset, err)
			continue
		}

		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Printf("eventlog: commit error on %s offset %d: %v", topic, msg.Offset, err)
		}
	}
}
