# Matching Engine

Central order-matching service for FluxMesh DEX. It consumes new orders from Kafka, runs price-time priority matching against per-market in-memory order books, and emits fill or rejection events back to Kafka for downstream services (indexer, settlement) to consume.

## Architecture

```
Kafka                     Matching Engine                    Kafka
──────                    ───────────────                    ─────
                          ┌──────────────┐
orders.created ─────────▶ │   Engine     │
                          │  ┌────────┐  │ ──▶ orders.matched   (per fill)
                          │  │  Book  │  │
                          │  │ (bids) │  │
                          │  │ (asks) │  │ ──▶ orders.rejected  (validation failure)
                          │  └────────┘  │
                          └──────────────┘
                           one book per market
```

**Flow:**

1. Kafka reader pulls a message from `orders.created`.
2. `Engine.ProcessCreated` validates fields (side, price, size).
3. Invalid orders produce an `orders.rejected` event and the offset is committed.
4. Valid orders are fed to the per-market `OrderBook.MatchIncoming`.
5. Each fill generates an `orders.matched` event; any remaining size rests on the book.
6. On success the consumer commits the Kafka offset.

## Package Layout

```
matching-engine/
├── cmd/matching-engine/main.go       # Entry point, Kafka reader, signal handling
├── internal/
│   ├── engine/
│   │   └── engine.go                 # Engine coordinator, event types, Kafka producer
│   └── orderbook/
│       └── book.go                   # OrderBook interface + price-time implementation
└── go.mod
```

| Package | Responsibility |
|---------|---------------|
| `cmd/matching-engine` | Bootstrap: Kafka consumer loop, graceful shutdown via `SIGINT`/`SIGTERM`. |
| `internal/engine` | Deserialize `orders.created`, validate, delegate to order book, publish results. Houses `KafkaProducer` and `EventProducer` interface. |
| `internal/orderbook` | Pure matching logic. Defines `Order`, `Fill`, `Side` types and the `OrderBook` interface with a `priceTimeOrderBook` implementation. |

## Order Book Design

| Aspect | Detail |
|--------|--------|
| **Priority** | Price-time (FIFO within the same price level) |
| **Storage** | In-memory slices (`bids` / `asks`) per market |
| **Isolation** | One `OrderBook` instance per `market_id`, created lazily with double-checked locking |
| **Bids ordering** | Descending by price, ascending by `CreatedAt` |
| **Asks ordering** | Ascending by price, ascending by `CreatedAt` |

```
bids []             asks []
─────────           ─────────
100.50  t1          101.00  t2
100.50  t3          101.50  t4
 99.00  t5          102.00  t6
   ▲                   ▲
best bid            best ask
```

## Matching Algorithm

1. Incoming **buy** order walks the `asks` slice (lowest price first).
   - If `incoming.Price >= ask.Price`, fill `min(incoming.Remaining, ask.Remaining)`.
   - Repeat until the incoming order is fully filled or no more crossable asks remain.
2. Incoming **sell** order walks the `bids` slice (highest price first), same logic reversed.
3. Fully filled resting orders are pruned from the slice.
4. If the incoming order still has remaining size after matching, it is added (rested) onto the book.

Fills execute at the **maker's price** (the resting order's price).

## Kafka Topics

| Direction | Topic | Payload | Notes |
|-----------|-------|---------|-------|
| **Consumed** | `orders.created` | `OrdersCreatedEvent` — order_id, user_id, market_id, side, type, price, size | Consumer group: `matching-engine` |
| **Produced** | `orders.matched` | trade_id, market_id, maker_order_id, taker_order_id, price, size, maker_side, ts | One message per fill |
| **Produced** | `orders.rejected` | order_id, user_id, market_id, reason, ts | Emitted for invalid side / price / size |

## Known Limitations

| Area | Limitation |
|------|-----------|
| **Precision** | Prices and sizes use `shopspring/decimal` for arbitrary-precision arithmetic — no float rounding. |
| **Persistence** | Order books are fully in-memory. A restart loses all resting orders. |
| **Cancellations** | No cancel/amend support — orders can only be added. |
| **Order types** | Only limit orders are matched; no market or stop orders. |
| **Concurrency** | Books are lazily created under a mutex, but `MatchIncoming` itself is not concurrency-safe per book (single consumer). |
| **Sorting** | Slices are re-sorted on every match call; fine for a skeleton, not for high-throughput production. |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker addresses |

## Running

```bash
cd matching-engine && go mod tidy && go run ./cmd/matching-engine
```

Requires a running Kafka broker with the `orders.created` topic available.
