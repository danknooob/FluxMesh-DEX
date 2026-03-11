package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/danknooob/fluxmesh-dex/matching-engine/internal/engine"
	"github.com/segmentio/kafka-go"
)

func main() {
	brokers := []string{"localhost:9092"}
	if env := os.Getenv("KAFKA_BROKERS"); env != "" {
		// simplistic split: "host1:9092,host2:9092"
		parts := strings.Split(env, ",")
		if len(parts) > 0 {
			brokers = parts
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	prod := engine.NewKafkaProducer(brokers, "orders.matched", "orders.rejected")
	eng := engine.NewEngine(prod)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    "orders.created",
		GroupID:  "matching-engine", // stable group id so committed offsets are not re-read
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	log.Println("matching-engine: consuming from topic orders.created")

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("matching-engine: shutting down")
				return
			}
			log.Printf("matching-engine: read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var evt engine.OrdersCreatedEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("matching-engine: failed to unmarshal event: %v", err)
			continue
		}
		if err := eng.ProcessCreated(ctx, evt); err != nil {
			log.Printf("matching-engine: failed to process event: %v", err)
			// Do not commit offset so that the message can be retried.
			time.Sleep(time.Second)
			continue
		}

		if err := reader.CommitMessages(ctx, m); err != nil {
			log.Printf("matching-engine: failed to commit message: %v", err)
			time.Sleep(time.Second)
			continue
		}
	}
}

