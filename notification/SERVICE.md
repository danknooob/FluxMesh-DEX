# Notification Service

Real-time WebSocket push server that bridges Kafka domain events to connected browser clients — delivering per-user notifications with zero polling.

## Purpose

Front-end clients open a persistent WebSocket connection to this service. On the back end, a Kafka consumer reads from `notifications.user`, extracts the `user_id` from each message payload, and pushes the raw JSON to every WebSocket connection belonging to that user. This decouples the API/event producers from the delivery mechanism.

## Architecture

```
Kafka                    Notification Service                  Browser
──────                   ────────────────────                  ───────
                         ┌──────────────────┐
notifications.user ────▶ │ NotificationConsumer │
                         └────────┬─────────┘
                                  │ hub.Broadcast(user_id, data)
                                  ▼
                         ┌──────────────────┐      WS frames
                         │       Hub        │ ──────────────▶  Client A (user X)
                         │  (fan-out by     │ ──────────────▶  Client B (user X)
                         │   user_id)       │ ──────────────▶  Client C (user Y)
                         └──────────────────┘
                                  ▲
                         ┌────────┴─────────┐
                         │   WSHandler      │ ◀──── HTTP Upgrade /ws?user_id=X
                         └──────────────────┘
```

### Data Flow

1. **Kafka consumer** reads from `notifications.user` (consumer group `notification-notifications-user`).
2. Each message is JSON-decoded; the `user_id` field is extracted.
3. `Hub.Broadcast` enqueues the raw JSON byte slice for every `Client` registered under that user ID.
4. Each client's writer goroutine drains its `Send` channel and writes WS text frames.
5. If a client's send buffer is full (256 messages) the message is dropped with a log warning.

## Kafka Topics

| Direction | Topic | Consumer Group | Description |
|-----------|-------|----------------|-------------|
| **Consumed** | `notifications.user` | `notification-notifications-user` | Per-user notification payloads (must contain `user_id`) |

Messages **without** a `user_id` field are logged and dropped.

## WebSocket Endpoint

| Method | Path | Query Params | Description |
|--------|------|--------------|-------------|
| GET (Upgrade) | `/ws` | `user_id` (required) | Upgrades to WebSocket; registers client in the Hub |

- **Protocol:** Standard WebSocket (RFC 6455) via `gorilla/websocket`.
- **Origin check:** All origins accepted (`CheckOrigin` returns `true`).
- **Read loop:** The server reads incoming frames only to detect connection close; no client-to-server messages are processed.
- **Write loop:** A dedicated goroutine per client writes queued messages from the `Send` channel.

## Package Layout

```
notification/
├── cmd/notification/main.go          # Entry point, wiring, HTTP server
├── internal/
│   ├── hub/
│   │   └── hub.go                    # Hub + Client types, fan-out event loop
│   ├── kafka/
│   │   └── consumer.go               # NotificationConsumer (Kafka → Hub)
│   └── server/
│       └── ws.go                     # WSHandler (HTTP upgrade → Hub registration)
└── go.mod
```

| Package | Responsibility |
|---------|---------------|
| `cmd/notification` | Bootstrap: create Hub, start consumer goroutine, bind HTTP mux, listen |
| `internal/hub` | Manages per-user client sets; broadcasts messages via channels |
| `internal/kafka` | Kafka reader for `notifications.user`; extracts `user_id` and forwards to Hub |
| `internal/server` | HTTP → WebSocket upgrade; registers/unregisters clients on the Hub |

### Key Types

| Type | Role |
|------|------|
| `Hub` | Central registry of connected clients; routes messages by `user_id` |
| `Client` | Single WebSocket connection; carries `UserID` and a buffered `Send` channel |
| `Message` | Internal struct: `{UserID, Data}` queued for broadcast |
| `NotificationConsumer` | Kafka reader that pushes raw payloads into the Hub |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker addresses |
| `NOTIFICATION_PORT` | `8090` | HTTP/WS listen port |

## Running

```bash
cd notification && go mod tidy && go run ./cmd/notification
```

Connect with any WebSocket client:

```
ws://localhost:8090/ws?user_id=alice
```

## Known Limitations

- **No JWT/auth** — `user_id` is taken from a query parameter; any caller can impersonate any user.
- **Single topic** — only `notifications.user` is consumed; other domain topics (`orders.matched`, `balances.updated`, etc.) are not yet wired.
- **No heartbeat/ping** — the server does not send WebSocket ping frames; idle connections may be killed by proxies.
- **No TLS** — plaintext HTTP/WS only; a reverse proxy is expected in production.
- **Drop-on-slow** — if a client's 256-message buffer fills up, subsequent messages are silently dropped.
