# System Architecture: Data Plane vs Control Plane

## Data Plane (Runtime Path)

Services that move orders and trades:

1. **MVC API** — HTTP gateway. Validates and persists orders, publishes `orders.created` to Kafka.
2. **Matching Engine** — Consumes `orders.created`, maintains in-memory order books, emits `orders.matched` / `orders.rejected`.
3. **Settlement** — Consumes `orders.matched`, batches and calls EVM `ExchangeCore.settleTrades`, emits `trades.settled` and `balances.updated`.
4. **Indexer** — Listens to chain events and `trades.settled`; updates Postgres read models (positions, balances, trade history).
5. **Notification** — Consumes domain + `notifications.user`; holds WebSocket connections and pushes updates to clients.

## Control Plane (MCP)

Manages the system like a cockpit:

- **Configuration & desired state** — Markets, tick size, fees, risk limits, feature flags. Stored in MCP DB; changes published to `control.config`.
- **Service registry and health** — Services heartbeat to MCP (HTTP/gRPC or `control.health`). MCP aggregates and exposes health in admin UI.
- **Access and audit** — Who can change markets/risk; all changes logged to `control.audit`.
- **Operational commands** — e.g. “pause market ETH/USDC”, “safe mode”. MCP writes to DB + `control.commands`; data-plane services subscribe and apply.

## Diagram

```
                    ┌──────────────────────────────────────┐
                    │           MCP Control Plane           │
                    │  control.config │ control.health      │
                    │  control.audit │ control.commands    │
                    └──────────────────────────────────────┘
                                         │
        ┌────────────────────────────────┼────────────────────────────────┐
        ▼                                ▼                                ▼
┌───────────────┐              ┌─────────────────┐              ┌─────────────────┐
│   MVC API     │──orders.created──▶│ Matching Engine │──orders.matched──▶│   Settlement   │
│  (Gateway)    │              │  (Order books)   │              │  (EVM settle)   │
└───────────────┘              └─────────────────┘              └────────┬────────┘
        │                                │                                │
        ▼                                ▼                                ▼
   Postgres                         orders.rejected              trades.settled
        │                                │                        balances.updated
        └────────────────────────────────┼────────────────────────────┘
                                         ▼
                                ┌─────────────────┐
                                │    Indexer      │
                                │ (Read models)   │
                                └────────┬────────┘
                                         │
        ┌────────────────────────────────┼────────────────────────────────┐
        ▼                                ▼                                ▼
┌───────────────┐              ┌─────────────────┐              ┌─────────────────┐
│  Notification │◀────────────│  Kafka topics   │─────────────▶│   Trader UI     │
│  (WebSocket)  │              │  (event log)    │              │   (React)       │
└───────────────┘              └─────────────────┘              └─────────────────┘
```
