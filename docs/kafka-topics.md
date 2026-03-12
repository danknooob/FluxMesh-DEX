# Kafka Topic Design

## Data Plane

| Topic | Producer | Consumers | Purpose |
|-------|----------|-----------|---------|
| `orders.created` | API | Matching engine, Event log | New limit/market orders |
| `orders.cancelled` | API | Matching, Indexer, Event log | Order cancellations |
| `orders.matched` | Matching engine | Settlement, Indexer, Notification, Event log | Fills and remaining size |
| `orders.rejected` | Matching engine | Indexer, Notification, Event log | Failed risk/validation |
| `trades.settled` | Settlement | Indexer, Notification, Event log | On-chain settlement done |
| `balances.updated` | Settlement | Indexer, Notification, Event log | Balance changes |
| `users.updated` | API | Event log | Profile name/email/avatar changes |
| `users.deleted` | API | Event log | Account soft-deletion |
| `notifications.user` | Various | Notification service, Event log | User-targeted notifications |

## Control Plane

| Topic | Producer | Consumers | Purpose |
|-------|----------|-----------|---------|
| `control.config` | Control plane | All data-plane services, Event log | Config, feature flags, markets |
| `control.health` | Data-plane services | Control plane, Event log | Heartbeats / health |
| `control.audit` | Control plane | Event log | Immutable audit log |
| `control.commands` | Control plane | Data-plane services, Event log | Pause market, safe mode, etc. |

## Consumer Groups

| Service | Group ID | Topics |
|---------|----------|--------|
| Matching engine | `matching-engine` | `orders.created` |
| Settlement | `settlement` | `orders.matched` |
| Indexer | `indexer` | `orders.matched`, `orders.rejected`, `trades.settled`, `balances.updated` |
| Notification | `notification` | Domain topics + `notifications.user` |
| Event log | `eventlog` | All 13 topics |
| Data-plane services | Per-service | `control.config`, `control.commands` |

## Offset Management

- **Matching engine & Settlement**: Explicit `FetchMessage` + `CommitMessages` — only commits after successful processing.
- **Indexer**: `FetchMessage` + retry with exponential backoff (3 retries, 300ms base) + `CommitMessages` after success. Drops and logs after exhausting retries.
- **Event log**: Only commits after successful MongoDB write, ensuring no events are lost.
- **Notification**: `ReadMessage` with auto-commit (at-most-once is acceptable for notifications).

## Event Log → MongoDB

Every message from every topic is persisted to MongoDB (`fluxmesh_events.events`) by the Event Log service. Each document includes:

| Field | Description |
|-------|-------------|
| `topic` | Kafka topic name |
| `title` | Human-readable event title (e.g. "New BTC-USDC order: buy 0.01 @ 62000", "Profile updated: alice@example.com changed name") |
| `key` | Kafka message key |
| `payload` | Parsed JSON payload as a BSON document |
| `offset` | Kafka offset |
| `partition` | Kafka partition |
| `timestamp` | Original Kafka message timestamp |
| `stored_at` | When the event was written to MongoDB |

### Title Examples

| Topic | Example Title |
|-------|---------------|
| `orders.created` | New BTC-USDC order: buy 0.01 @ 62000 |
| `orders.matched` | Order matched: abc123 |
| `orders.rejected` | Order rejected: abc123 |
| `trades.settled` | Trade settled: abc123 |
| `balances.updated` | Balance updated for user xyz |
| `users.updated` | Profile updated: alice@example.com changed name, email |
| `users.deleted` | Account deleted: user-uuid |
| `control.config` | Config change |
| `control.health` | Health heartbeat |
| `control.audit` | Audit entry |
| `control.commands` | Command issued |
