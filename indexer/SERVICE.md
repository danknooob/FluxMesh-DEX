# Indexer Service

Kafka consumer that projects events into Postgres read models — closing the loop between async event processing and queryable state.

## Purpose

When a user places an order, the API persists it as `pending` and publishes to Kafka. The matching engine, settlement, and balance services process asynchronously. The **Indexer** is the service that writes the results back to Postgres so the API can serve up-to-date order statuses, trade history, and balances.

```
Kafka                        Indexer                    Postgres
──────                       ───────                    ────────
orders.matched   ──────────▶ OrderMatchedHandler   ──▶  orders (status update)
                                                   ──▶  trades (new row)
orders.rejected  ──────────▶ OrderRejectedHandler  ──▶  orders (status → rejected)
trades.settled   ──────────▶ TradeSettledHandler    ──▶  trades (settled_at timestamp)
balances.updated ──────────▶ BalanceUpdatedHandler  ──▶  balances (upsert available/locked)
```

## Low-Level Design

### SOLID Principles Applied

| Principle | How |
|-----------|-----|
| **Single Responsibility** | Each `EventHandler` handles exactly one Kafka topic. Repositories handle only DB access. The dispatcher only routes messages. |
| **Open/Closed** | New event types are added by creating a new handler and registering it in the dispatcher — zero changes to existing code. |
| **Liskov Substitution** | Every handler satisfies the `EventHandler` interface and is interchangeable in the dispatcher's handler map. |
| **Interface Segregation** | Small, focused interfaces: `OrderWriter` (update status), `TradeWriter` (create/settle), `BalanceWriter` (upsert). No handler depends on methods it doesn't use. |
| **Dependency Inversion** | Handlers depend on repository *interfaces*, not GORM directly. `main.go` wires concrete implementations via constructor injection. |

### Package Layout

```
indexer/
├── cmd/indexer/main.go            # Entry point, DI wiring, graceful shutdown
├── internal/
│   ├── model/
│   │   ├── event.go               # Kafka event payload structs (typed deserialization)
│   │   └── trade.go               # Trade DB model (maps to `trades` table)
│   ├── repository/
│   │   ├── interfaces.go          # OrderWriter, TradeWriter, BalanceWriter interfaces
│   │   ├── order_repo.go          # GORM OrderWriter implementation
│   │   ├── trade_repo.go          # GORM TradeWriter implementation
│   │   └── balance_repo.go        # GORM BalanceWriter implementation
│   ├── handler/
│   │   ├── handler.go             # EventHandler interface + HandlerRegistry
│   │   ├── order_matched.go       # orders.matched → update order status + create trade
│   │   ├── order_rejected.go      # orders.rejected → update order status to rejected
│   │   ├── trade_settled.go       # trades.settled → set settled_at on trade
│   │   └── balance_updated.go     # balances.updated → upsert balance row
│   └── consumer/
│       └── dispatcher.go          # Kafka consumer per topic, routes to handler
└── go.mod

```

### Class Diagram (Interfaces)

```
                    ┌──────────────────┐
                    │  EventHandler    │ (interface)
                    │  Handle(ctx, []byte) error │
                    └────────┬─────────┘
           ┌────────────┬────┴────────┬──────────────┐
           ▼            ▼             ▼              ▼
  OrderMatchedHandler  OrderRejected  TradeSettled  BalanceUpdated
   │                    Handler       Handler       Handler
   │                      │             │              │
   ▼                      ▼             ▼              ▼
 OrderWriter           OrderWriter   TradeWriter   BalanceWriter
 TradeWriter            (interface)  (interface)   (interface)
 (interfaces)
```

### Sequence: Order Matched

```
Kafka msg (orders.matched) → Dispatcher → OrderMatchedHandler
  1. Deserialize payload → OrderMatchedEvent
  2. Call OrderWriter.UpdateStatus(makerOrderID, "matched"/"partial")
  3. Call OrderWriter.UpdateStatus(takerOrderID, "matched"/"partial")
  4. Call TradeWriter.Create(trade)
  5. Return nil (dispatcher commits offset)
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DB_DSN` | `postgres://dex:dex@localhost:5432/fluxmesh?sslmode=disable` | Postgres connection string |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses |
| `INDEXER_PORT` | `8082` | HTTP port (health endpoint) |

## Running

```bash
cd indexer && go mod tidy && go run ./cmd/indexer
```
