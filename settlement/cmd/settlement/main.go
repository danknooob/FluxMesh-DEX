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

	"github.com/danknooob/fluxmesh-dex/settlement/internal/engine"
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

	prod := engine.NewKafkaProducer(brokers, "trades.settled", "balances.updated")
	eng := engine.NewEngine(prod)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    "orders.matched",
		GroupID:  "settlement", // stable group id
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer func() { _ = reader.Close() }()

	log.Println("settlement: consuming from topic orders.matched")

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("settlement: shutting down")
				return
			}
			log.Printf("settlement: read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var t engine.MatchedTrade
		if err := json.Unmarshal(m.Value, &t); err != nil {
			log.Printf("settlement: failed to unmarshal matched trade: %v", err)
			_ = reader.CommitMessages(ctx, m)
			continue
		}

		if err := eng.ProcessMatched(ctx, t); err != nil {
			log.Printf("settlement: failed to process matched trade: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if err := reader.CommitMessages(ctx, m); err != nil {
			log.Printf("settlement: failed to commit message: %v", err)
			time.Sleep(time.Second)
		}
	}
}

