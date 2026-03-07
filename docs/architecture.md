# System Architecture: Data Plane vs Control Plane

## Data Plane (Runtime Path)

Services that move orders and trades:

1. **MVC API** — HTTP gateway. Validates and persists orders, publishes `orders.created` to Kafka.
2. **Matching Engine** — Consumes `orders.created`, maintains in-memory order books, emits `orders.matched` / `orders.rejected`.
3. **Settlement** — Consumes `orders.matched`, batches and calls EVM `ExchangeCore.settleTrades`, emits `trades.settled` and `balances.updated`.
4. **Indexer** — Listens to chain events and `trades.settled`; updates Postgres read models (positions, balances, trade history).
5. **Notification** — Consumes domain + `notifications.user`; holds WebSocket connections and pushes updates to clients.

## Control Plane

Manages the system like a cockpit:

- **Configuration & desired state** — Markets, tick size, fees, risk limits, feature flags. Stored in control-plane DB; changes published to `control.config`.
- **Service registry and health** — Services heartbeat (HTTP/gRPC or `control.health`). Control plane aggregates and exposes health in admin UI.
- **Access and audit** — Who can change markets/risk; all changes logged to `control.audit`.
- **Operational commands** — e.g. “pause market ETH/USDC”, “safe mode”. Control plane writes to DB + `control.commands`; data-plane services subscribe and apply.

## MCP (Model Context Protocol)

A separate MCP server exposes DEX capabilities to AI assistants (e.g. Cursor, Claude): tools such as `get_markets`, `get_balances`, `get_health` so AI can query the exchange without custom integrations.

## Diagram

```
                    ┌──────────────────────────────────────┐
                    │     Control Plane + MCP server        │
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
