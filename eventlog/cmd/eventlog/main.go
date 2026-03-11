package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/danknooob/fluxmesh-dex/eventlog/internal/consumer"
	"github.com/danknooob/fluxmesh-dex/eventlog/internal/store"
)

var defaultTopics = []string{
	"orders.created",
	"orders.cancelled",
	"orders.matched",
	"orders.rejected",
	"trades.settled",
	"balances.updated",
	"notifications.user",
	"control.config",
	"control.health",
	"control.audit",
	"control.commands",
}

func main() {
	mongoURI := getEnv("MONGO_URI", "mongodb://fluxmesh:fluxmesh_secret@localhost:27017")
	mongoDB := getEnv("MONGO_DB", "fluxmesh_events")
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	groupID := getEnv("KAFKA_GROUP_ID", "eventlog")

	topicsEnv := os.Getenv("KAFKA_TOPICS")
	topics := defaultTopics
	if topicsEnv != "" {
		topics = strings.Split(topicsEnv, ",")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	es, err := store.NewMongoStore(ctx, mongoURI, mongoDB)
	if err != nil {
		log.Fatalf("eventlog: mongo connection failed: %v", err)
	}
	defer es.Close(ctx)

	log.Printf("eventlog: connected to MongoDB (%s/%s)", mongoURI, mongoDB)
	log.Printf("eventlog: subscribing to %d topics: %v", len(topics), topics)

	tc := consumer.New(brokers, groupID, topics, es)

	go tc.Run(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("eventlog: shutting down")
	cancel()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
