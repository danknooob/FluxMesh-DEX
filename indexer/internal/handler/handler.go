package handler

import "context"

// EventHandler processes a single Kafka message payload.
// Each implementation handles exactly one topic (Single Responsibility).
type EventHandler interface {
	Handle(ctx context.Context, payload []byte) error
}

// Registry maps Kafka topic names to their handlers.
// Adding a new topic requires only a new handler and one registry entry
// — existing handlers are never modified (Open/Closed).
type Registry map[string]EventHandler
