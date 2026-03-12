package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/danknooob/fluxmesh-dex/notification/internal/hub"
	notifyKafka "github.com/danknooob/fluxmesh-dex/notification/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/notification/internal/server"
)

func main() {
	brokers := []string{"localhost:9092"}
	if env := os.Getenv("KAFKA_BROKERS"); env != "" {
		parts := strings.Split(env, ",")
		if len(parts) > 0 {
			brokers = parts
		}
	}

	h := hub.NewHub()
	go h.Run()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	userCons := notifyKafka.NewNotificationConsumer(brokers, "notifications.user", h)
	go userCons.Run(ctx)

	matchedCons := notifyKafka.NewMatchedConsumer(brokers, h)
	go matchedCons.Run(ctx)

	cancelCons := notifyKafka.NewTypedConsumer(brokers, "orders.cancelled", "order_cancelled", h)
	go cancelCons.Run(ctx)

	balanceCons := notifyKafka.NewTypedConsumer(brokers, "balances.updated", "balance_updated", h)
	go balanceCons.Run(ctx)

	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.WSHandler(h, []byte(jwtSecret)))

	addr := ":8090"
	if env := os.Getenv("NOTIFICATION_PORT"); env != "" {
		addr = ":" + env
	}

	log.Printf("notification: WebSocket server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("notification: server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

