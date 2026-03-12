package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/danknooob/fluxmesh-dex/matching-engine/internal/engine"
	"github.com/segmentio/kafka-go"
)

func main() {
	brokers := []string{"localhost:9092"}
	if env := os.Getenv("KAFKA_BROKERS"); env != "" {
		parts := strings.Split(env, ",")
		if len(parts) > 0 {
			brokers = parts
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	prod := engine.NewKafkaProducer(brokers, "orders.matched", "orders.rejected")
	eng := engine.NewEngine(prod)

	createdReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    "orders.created",
		GroupID:  "matching-engine",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer createdReader.Close()

	cancelledReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    "orders.cancelled",
		GroupID:  "matching-engine",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer cancelledReader.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		consumeCreated(ctx, createdReader, eng)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		consumeCancelled(ctx, cancelledReader, eng)
	}()

	log.Println("matching-engine: consuming orders.created + orders.cancelled")
	wg.Wait()
	log.Println("matching-engine: shutdown complete")
}

func consumeCreated(ctx context.Context, r *kafka.Reader, eng *engine.Engine) {
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("matching-engine: read error (created): %v", err)
			time.Sleep(time.Second)
			continue
		}

		var evt engine.OrdersCreatedEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("matching-engine: unmarshal error (created): %v", err)
			_ = r.CommitMessages(ctx, m)
			continue
		}

		if err := eng.ProcessCreated(ctx, evt); err != nil {
			log.Printf("matching-engine: process created error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if err := r.CommitMessages(ctx, m); err != nil {
			log.Printf("matching-engine: commit error (created): %v", err)
		}
	}
}

func consumeCancelled(ctx context.Context, r *kafka.Reader, eng *engine.Engine) {
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("matching-engine: read error (cancelled): %v", err)
			time.Sleep(time.Second)
			continue
		}

		var evt engine.OrdersCancelledEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("matching-engine: unmarshal error (cancelled): %v", err)
			_ = r.CommitMessages(ctx, m)
			continue
		}

		eng.ProcessCancelled(ctx, evt)

		if err := r.CommitMessages(ctx, m); err != nil {
			log.Printf("matching-engine: commit error (cancelled): %v", err)
		}
	}
}
