# Eventlog Service

Universal Kafka-to-MongoDB audit trail that captures every domain and control-plane event into a queryable, indexed document store — providing a full history of everything that happens on the exchange.

## Purpose

Every Kafka topic in FluxMesh DEX carries events that are important for auditing, debugging, and analytics. The **Eventlog** service subscribes to *all* of them (12 topics by default), deserializes each message, generates a human-readable **title**, and persists a structured document into MongoDB. The result is a single `events` collection that serves as the system-of-record audit trail.

```
Kafka (12 topics)           Eventlog                    MongoDB
─────────────────           ────────                    ───────
orders.created    ─────┐
orders.cancelled  ─────┤
orders.matched    ─────┤
orders.rejected   ─────┤    ┌──────────────────┐
trades.settled    ─────┤───▶│  TopicConsumer    │──────▶  events collection
balances.updated  ─────┤    │  (1 reader/topic) │       (with retry)
notifications.user─────┤    └──────────────────┘
users.updated     ─────┤
users.deleted     ─────┤
control.config    ─────┤
control.health    ─────┤
control.audit     ─────┤
control.commands  ─────┘
```

## Topics Consumed

| Topic | Consumer Group | Title Example |
|-------|---------------|---------------|
| `orders.created` | `eventlog` | `New ETH-USDC order: buy 1.5 @ 3200` |
| `orders.cancelled` | `eventlog` | `Order cancelled: ord_abc123` |
| `orders.matched` | `eventlog` | `Order matched on ETH-USDC (3 fills)` |
| `orders.rejected` | `eventlog` | `Order rejected: ord_xyz — insufficient balance` |
| `trades.settled` | `eventlog` | `Trades settled on ETH-USDC (batch trades)` |
| `balances.updated` | `eventlog` | `Balance updated: user alice, asset USDC` |
| `notifications.user` | `eventlog` | `Notification [fill] for user alice` |
| `users.updated` | `eventlog` | `Profile updated: user alice changed email to ...` |
| `users.deleted` | `eventlog` | `Account deleted: alice@example.com (user_42)` |
| `control.config` | `eventlog` | `Config change: update ETH-USDC` |
| `control.health` | `eventlog` | `Health heartbeat: api is healthy` |
| `control.audit` | `eventlog` | `Audit: disable_market by admin_01` |
| `control.commands` | `eventlog` | `Command: restart → matching-engine` |

All readers share the same `groupID` (`eventlog` by default), so Kafka distributes partitions across instances if you scale horizontally.

## MongoDB Document Structure

Each event is stored in the `events` collection with this shape:

```json
{
  "topic":     "orders.created",
  "title":     "New ETH-USDC order: buy 1.5 @ 3200",
  "key":       "<Kafka message key, if any>",
  "payload":   { /* original JSON body as BSON map */ },
  "offset":    42,
  "partition":  0,
  "timestamp": "2026-03-12T10:30:00Z",
  "stored_at": "2026-03-12T10:30:00.123Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `topic` | string | Kafka topic the event came from |
| `title` | string | Auto-generated human-readable summary (see `TitleForEvent`) |
| `key` | string | Kafka message key (empty for unkeyed messages) |
| `payload` | object | Full event body deserialized as BSON; falls back to `{"raw": "..."}` if not valid JSON |
| `offset` | int64 | Kafka partition offset |
| `partition` | int | Kafka partition number |
| `timestamp` | datetime | Kafka message timestamp (broker or producer time) |
| `stored_at` | datetime | UTC wall-clock time when the document was written to MongoDB |

### Indexes

| Index | Purpose |
|-------|---------|
| `{topic: 1, timestamp: -1}` | Efficient per-topic chronological queries |
| `{title: 1}` | Fast title-based search / filtering |
| `{stored_at: 1}` | TTL-ready; enables "last N minutes" queries |

### Title Generation

The `TitleForEvent` function inspects the topic name and extracts key fields from the payload to produce a concise one-line summary. Each of the 12 default topics has a dedicated formatting branch; unknown topics fall back to `"Event on <topic>"`.

## Retry Strategy

MongoDB write failures are retried with **exponential back-off + jitter**:

| Parameter | Value |
|-----------|-------|
| Max retries | 4 |
| Base delay | 300 ms |
| Max delay cap | 10 s |
| Jitter | random `[0, 300ms)` added to each delay |
| On exhaustion | Event is **dropped** with an error log; offset is still committed |

```
attempt 0 → immediate
attempt 1 → ~600ms  + jitter
attempt 2 → ~1.2s   + jitter
attempt 3 → ~2.4s   + jitter
attempt 4 → ~4.8s   + jitter   (or capped at 10s)
  └─ give up, log "DROPPING event", commit offset
```

## Package Layout

```
eventlog/
├── cmd/eventlog/main.go              # Entry point, env parsing, topic list, signal handling
├── internal/
│   ├── consumer/
│   │   └── consumer.go               # TopicConsumer: one kafka.Reader per topic, retry logic
│   └── store/
│       └── mongo.go                  # MongoStore, EventDocument, ParsePayload, TitleForEvent
└── go.mod
```

| Package | Responsibility |
|---------|---------------|
| `cmd/eventlog` | Bootstrap: connect to MongoDB, resolve topic list, start `TopicConsumer`, await SIGINT/SIGTERM |
| `internal/consumer` | Creates one `kafka.Reader` per topic; runs a goroutine per reader with `saveWithRetry` |
| `internal/store` | MongoDB connection, `EventStore` interface, document shape, JSON→BSON parsing, title generation |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker addresses |
| `KAFKA_GROUP_ID` | `eventlog` | Consumer group ID |
| `KAFKA_TOPICS` | *(all 12 default topics)* | Comma-separated override for subscribed topics |
| `MONGO_URI` | `mongodb://fluxmesh:fluxmesh_secret@localhost:27017` | MongoDB connection URI |
| `MONGO_DB` | `fluxmesh_events` | MongoDB database name |

## Running

```bash
cd eventlog && go mod tidy && go run ./cmd/eventlog
```

To subscribe to a subset of topics:

```bash
KAFKA_TOPICS=orders.created,trades.settled go run ./cmd/eventlog
```
