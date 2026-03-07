# FluxMesh DEX — Event-Driven Order-Book DEX

Production-grade, event-driven order-book DEX backend with Kafka data plane and an MCP (Microservice Control Plane) for config, health, and operations.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CONTROL PLANE (MCP)                                │
│  Config • Health • Feature flags • Audit • Admin API                         │
│  Topics: control.config, control.health, control.audit, control.commands     │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            DATA PLANE                                        │
│  orders.created → Matching Engine → orders.matched → Settlement → EVM        │
│       ↓                    ↓                         ↓                      │
│  Postgres            orders.rejected           trades.settled                │
│       ↓                    ↓                         ↓                      │
│  Indexer ←────────── balances.updated ←──────────────┘                       │
│       ↓                                                                      │
│  Notification / WebSocket → Trader UI                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

- **Data plane**: MVC API, matching engine, settlement, indexer, notification service — all over Kafka + Postgres.
- **Control plane (MCP)**: Configuration, service registry/health, audit, and operational commands.

## Repo Layout

| Path | Description |
|------|-------------|
| `contracts/` | EVM smart contracts (ExchangeCore, MarketRegistry) |
| `api/` | Go MVC HTTP gateway (controllers, services, repositories, Kafka producer) |
| `matching-engine/` | Order-book matching; consumes `orders.created`, emits `orders.matched` / `orders.rejected` |
| `settlement/` | Consumes `orders.matched`, batches and calls EVM `ExchangeCore.settleTrades` |
| `indexer/` | Blockchain + `trades.settled` → Postgres read models |
| `notification/` | WebSocket service; consumes domain + notification topics |
| `mcp/` | Control plane: config CRUD, health dashboard, control.* topics |
| `frontend/` | React — Trader UI + Admin UI for MCP |

## Kafka Topics

| Topic | Producer | Consumers | Purpose |
|-------|----------|-----------|---------|
| `orders.created` | API | Matching engine | New limit/market orders |
| `orders.cancelled` | API | Matching, Indexer | Order cancellations |
| `orders.matched` | Matching engine | Settlement, Indexer, Notification | Fills and remaining size |
| `orders.rejected` | Matching engine | Indexer, Notification | Failed risk/validation |
| `trades.settled` | Settlement | Indexer, Notification | On-chain settlement done |
| `balances.updated` | Settlement | Indexer, Notification | Balance changes |
| `notifications.user` | Various | Notification service | User-targeted notifications |
| `control.config` | MCP | All data-plane services | Config/feature flags |
| `control.health` | Data-plane services | MCP | Heartbeats/health |
| `control.audit` | MCP | — | Immutable audit log |
| `control.commands` | MCP | Data-plane services | Pause market, safe mode, etc. |

## Quick Start

1. **Infrastructure**
   ```bash
   docker-compose up -d
   ```
   Starts Kafka, Zookeeper, Postgres, and (optional) Redis for sessions.

2. **API (Go)**
   ```bash
   cd api && go mod tidy && go run ./cmd/api
   ```

3. **Frontend**
   ```bash
   cd frontend && npm install && npm run dev
   ```

4. **MCP control plane**
   ```bash
   cd mcp && go mod tidy && go run ./cmd/mcp
   ```

## Tradeoffs & Design Notes

- **Event-driven vs synchronous**: Orders are accepted via API and processed asynchronously via Kafka; clients get real-time updates via WebSocket. This improves throughput and decouples services.
- **Why Kafka**: Durable, ordered event log; replay and multiple consumers; aligns with control plane broadcasting config/commands.
- **What MCP controls**: Markets (enabled/params), risk limits, feature flags, service health view, audit trail, and operational commands (e.g. pause market, safe mode).

## Docs & Diagrams

- `docs/architecture.md` — Data plane vs control plane.
- `docs/sequence-order-lifecycle.md` — Order lifecycle.
- `docs/sequence-config-lifecycle.md` — Config change lifecycle.

## License

MIT
