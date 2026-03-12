# Settlement Service

Skeleton settlement engine that consumes matched trades from Kafka, simulates on-chain settlement, and publishes downstream events — bridging the matching engine to the balance layer.

## Purpose

After the matching engine pairs a maker and taker order, the resulting fill lands on the `orders.matched` Kafka topic. The **Settlement** service picks it up, would normally batch trades and call `ExchangeCore.settleTrades` on an EVM chain, and then emits two events: `trades.settled` (confirming the fill) and `balances.updated` (reflecting the new asset deltas for each counterparty).

> **Current status:** The engine is a **skeleton**. There are no actual Ethereum client calls, contract bindings, or batching logic. Every incoming matched trade is immediately "settled" in-memory and the downstream events are published as if the on-chain transaction succeeded.

```
Kafka                      Settlement Engine               Kafka
──────                     ─────────────────               ──────
orders.matched  ────────▶  ProcessMatched()  ──────────▶  trades.settled
                           (placeholder EVM call)  ────▶  balances.updated
```

## Kafka Topics

| Direction | Topic | Description |
|-----------|-------|-------------|
| **Consumed** | `orders.matched` | Single fill from the matching engine |
| **Produced** | `trades.settled` | Confirmation that a trade was settled (on-chain in the future) |
| **Produced** | `balances.updated` | Per-user per-asset balance deltas (placeholder payload) |

### `orders.matched` payload (input)

| Field | Type | Description |
|-------|------|-------------|
| `trade_id` | string | Unique fill identifier |
| `market_id` | string | Trading pair (e.g. `ETH-USDC`) |
| `maker_order_id` | string | Maker's order ID |
| `taker_order_id` | string | Taker's order ID |
| `price` | string | Fill price (decimal string) |
| `size` | string | Fill quantity (decimal string) |
| `maker_side` | string | `"buy"` or `"sell"` |
| `ts` | string | Match timestamp (RFC 3339) |

### `trades.settled` payload (output)

Same fields as input, with `price` and `size` converted from string to float64.

### `balances.updated` payload (output)

Placeholder: `{ "note": "...", "trade_id": "...", "market_id": "...", "ts": "..." }`. In production this would contain per-user per-asset signed deltas.

## Package Layout

```
settlement/
├── cmd/settlement/main.go        # Entry point, Kafka reader, graceful shutdown
├── internal/
│   └── engine/
│       └── engine.go             # MatchedTrade struct, Engine, KafkaProducer
└── go.mod
```

| Package | Responsibility |
|---------|---------------|
| `cmd/settlement` | Bootstrap: parse env, create Kafka reader (consumer group `settlement`), wire engine, signal handling |
| `internal/engine` | Core logic: `Engine.ProcessMatched` consumes a fill, emits events via `EventProducer` interface |

### Key Types

| Type | Role |
|------|------|
| `MatchedTrade` | Deserialized `orders.matched` payload |
| `SettlementBatch` | (Unused) future grouping of trades for a single EVM call |
| `EventProducer` | Interface — `PublishTradesSettled`, `PublishBalancesUpdated` |
| `KafkaProducer` | Concrete `EventProducer` backed by two `kafka.Writer` instances |
| `Engine` | Settlement logic; holds an `EventProducer` and a clock function |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker addresses |

## Running

```bash
cd settlement && go mod tidy && go run ./cmd/settlement
```

## Known Limitations

- **No EVM integration** — settlement is a no-op; trades are immediately marked as settled.
- **No batching** — each matched trade is processed one-by-one instead of being grouped into efficient on-chain batches.
- **No retry on produce failure** — if a `trades.settled` or `balances.updated` publish fails the message is not committed but there is no exponential back-off.
- **Balance deltas are stubs** — `balances.updated` payload does not contain real per-user asset changes.
