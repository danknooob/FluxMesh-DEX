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

	// For now we only consume notifications.user; later we can add
	// orders.matched, balances.updated, etc., and shape payloads per user.
	cons := notifyKafka.NewNotificationConsumer(brokers, "notifications.user", h)
	go cons.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.WSHandler(h))

	addr := ":8090"
	if env := os.Getenv("NOTIFICATION_PORT"); env != "" {
		addr = ":" + env
	}

	log.Printf("notification: WebSocket server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("notification: server error: %v", err)
	}
}

