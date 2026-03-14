package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/danknooob/fluxmesh-dex/indexer/internal/consumer"
	"github.com/danknooob/fluxmesh-dex/indexer/internal/handler"
	"github.com/danknooob/fluxmesh-dex/indexer/internal/model"
	"github.com/danknooob/fluxmesh-dex/indexer/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	topicOrdersMatched   = "orders.matched"
	topicOrdersRejected  = "orders.rejected"
	topicTradesSettled    = "trades.settled"
	topicBalancesUpdated  = "balances.updated"
)

func main() {
	dsn := getEnv("DB_DSN", "postgres://dex:dex@localhost:5432/fluxmesh?sslmode=disable")
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	port := getEnv("INDEXER_PORT", "8082")

	// --- Postgres ---
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("indexer: db connect: %v", err)
	}
	if err := db.AutoMigrate(&model.Trade{}); err != nil {
		log.Fatalf("indexer: migrate trades: %v", err)
	}
	log.Println("indexer: database connected, trades table migrated")

	// --- Repositories (concrete → interface) ---
	orderWriter := repository.NewOrderWriter(db)
	tradeWriter := repository.NewTradeWriter(db)
	balanceWriter := repository.NewBalanceWriter(db)

	// --- Handlers (depend on interfaces, not GORM) ---
	registry := handler.Registry{
		topicOrdersMatched:  handler.NewOrderMatchedHandler(orderWriter),
		topicOrdersRejected: handler.NewOrderRejectedHandler(orderWriter),
		topicTradesSettled:   handler.NewTradeSettledHandler(tradeWriter),
		topicBalancesUpdated: handler.NewBalanceUpdatedHandler(balanceWriter),
	}

	// --- Consumer dispatcher ---
	dispatcher := consumer.NewDispatcher(brokers, "indexer", registry)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Minimal health endpoint so orchestrators can probe liveness.
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		log.Printf("indexer: health endpoint on :%s/health", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil && err != http.ErrServerClosed {
			log.Printf("indexer: health server: %v", err)
		}
	}()

	log.Printf("indexer: starting (brokers=%v, topics=%d)", brokers, len(registry))
	dispatcher.Run(ctx)
	log.Println("indexer: shutdown complete")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
