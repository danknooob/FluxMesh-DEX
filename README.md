# FluxMesh DEX — Event-Driven Order-Book DEX

Production-grade, event-driven order-book DEX backend with an **API Gateway** (JWT auth + per-user rate limiting), Kafka data plane, **MongoDB event log**, a control plane for config/health/operations, and an **MCP (Model Context Protocol)** server so AI assistants can query markets, balances, and health.

## Architecture Overview

```
                        ┌───────────────────┐
                        │   React Frontend  │
                        │  (Trader + Admin) │
                        └─────────┬─────────┘
                                  │
                        ┌─────────▼─────────┐
                        │   API Gateway     │  :8000
                        │  ┌─────────────┐  │
                        │  │ JWT Auth     │  │  Validates every request
                        │  │ Rate Limiter │  │  Token-bucket per user/IP
                        │  └─────────────┘  │
                        └───┬───────────┬───┘
                            │           │
               ┌────────────▼──┐   ┌────▼──────────────┐
               │  API Service  │   │  Control Plane     │
               │  :8080        │   │  :8081             │
               │  Auth/Profile │   │  Config · Health   │
               │  Orders/Mkts  │   │  Audit · Commands  │
               │  Balances     │   └────────────────────┘
               └───────┬───────┘
                       │  Kafka
        ┌──────────────┼──────────────┬──────────────┐
        ▼              ▼              ▼              ▼
  ┌───────────┐ ┌───────────┐ ┌────────────┐ ┌────────────┐
  │ Matching  │ │Settlement │ │Notification│ │ Event Log  │
  │ Engine    │ │ Service   │ │ WebSocket  │ │ → MongoDB  │
  └───────────┘ └───────────┘ └────────────┘ └────────────┘
```

### Request Flow

```
Client → Gateway (JWT check + rate limit) → API Service → Postgres / Kafka
                                          → Control Plane (admin only)

Kafka topics → Event Log Service → MongoDB (immutable audit trail)
```

1. **Gateway** validates JWT, enforces per-user rate limits (token bucket), then injects `X-User-ID` / `X-Role` headers.
2. **API Service** trusts the gateway headers — no duplicate token parsing, keeping handler latency minimal.
3. **Functional APIs** (orders, markets, balances, profile) focus purely on business logic with zero auth overhead.
4. **Event Log** service consumes every Kafka topic (including `users.updated` and `users.deleted`) and persists events as documents in MongoDB for querying, debugging, and compliance.

## Repo Layout

| Path | Description |
|------|-------------|
| `gateway/` | API Gateway — JWT validation, per-user token-bucket rate limiting, reverse proxy ([SERVICE.md](gateway/SERVICE.md)) |
| `contracts/` | EVM smart contracts (ExchangeCore, MarketRegistry) with access control + per-trade idempotent settlement guards |
| `api/` | Go MVC HTTP service (auth, profile, orders, markets, balances, Kafka producer) ([SERVICE.md](api/SERVICE.md)) |
| `matching-engine/` | Order-book matching; restores resting orders from Postgres on startup; consumes `orders.created` and `orders.cancelled`; emits `orders.matched` / `orders.rejected` ([SERVICE.md](matching-engine/SERVICE.md)) |
| `indexer/` | Kafka → Postgres projector; updates order statuses, creates trade records, upserts balances ([SERVICE.md](indexer/SERVICE.md)) |
| `settlement/` | Consumes `orders.matched`, batches and calls EVM `ExchangeCore.settleTrades` ([SERVICE.md](settlement/SERVICE.md)) |
| `notification/` | WebSocket service; consumes domain + notification topics ([SERVICE.md](notification/SERVICE.md)) |
| `eventlog/` | Kafka → MongoDB event logger; subscribes to all topics and persists every event ([SERVICE.md](eventlog/SERVICE.md)) |
| `mcp/` | Control plane HTTP API + MCP (Model Context Protocol) server with DEX tools for AI ([SERVICE.md](mcp/SERVICE.md)) |
| `frontend/` | React — public Home, Trader UI, Admin UI |

## API Gateway

The gateway (`gateway/`) is the **single entry point** for all client traffic.

| Concern | Implementation |
|---------|---------------|
| **Authentication** | Validates `Authorization: Bearer <JWT>` on every protected route; admin routes additionally require `role=admin` |
| **Rate Limiting** | Token-bucket per authenticated user (20 req/s, burst 40). Falls back to per-IP for unauthenticated endpoints |
| **Header Injection** | After validation, injects `X-User-ID` and `X-Role` headers so downstream services skip token parsing |
| **Reverse Proxy** | Forwards to API (`:8080`) for trader routes, Control Plane (`:8081`) for admin routes |

### Why a separate gateway?

- **Latency**: Business-logic handlers never touch JWT crypto. Auth is done once at the edge.
- **Single Responsibility**: Rate limiting, auth, and routing are isolated from domain services.
- **Scalability**: Gateway can be scaled independently and replaced with an off-the-shelf solution (Kong, Envoy) later.

## Event Log (MongoDB)

The `eventlog/` service is a dedicated Kafka consumer that persists **every event** from all topics into MongoDB.

| Feature | Detail |
|---------|--------|
| **Topics consumed** | All 13 Kafka topics (orders, trades, balances, users, control, notifications) |
| **Storage** | MongoDB `fluxmesh_events.events` collection |
| **Document shape** | `{ topic, title, key, payload, offset, partition, timestamp, stored_at }` |
| **Human-readable titles** | Each event gets an auto-generated title (e.g. "Profile updated: alice@example.com changed name") |
| **Indexes** | Compound `(topic, timestamp)` for filtered queries; `title` for text search; `stored_at` for TTL/retention |
| **Offset management** | Explicit `FetchMessage` + `CommitMessages` — only commits after successful MongoDB write |
| **Use cases** | Audit trail, debugging, compliance, analytics, event replay |

### Why MongoDB for events?

- **Schema-flexible**: Different topics have different payload shapes. MongoDB handles this naturally without migrations.
- **Query-friendly**: Rich query language for filtering events by topic, time range, user, or payload fields.
- **Append-heavy workload**: MongoDB excels at high-throughput inserts, which matches the event log pattern.

## Kafka Topics

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
| `control.config` | Control plane | All data-plane services, Event log | Config/feature flags |
| `control.health` | Data-plane services | Control plane, Event log | Heartbeats/health |
| `control.audit` | Control plane | Event log | Immutable audit log |
| `control.commands` | Control plane | Data-plane services, Event log | Pause market, safe mode, etc. |

## WebSocket Notifications

The `notification/` service exposes a JWT-authenticated WebSocket endpoint used by the frontend to stream real-time events (fills, cancels, balance updates) per user.

- **Endpoint**: `ws://localhost:8090/ws?token=<jwt>`
- **Auth**: The token is the same JWT issued by the API; it is validated server-side using `JWT_SECRET` and the `sub` claim becomes the user id.
- **Topics bridged**: `notifications.user`, `orders.matched`, `orders.cancelled`, `balances.updated` → pushed to the correct user via an in-memory hub.
- **Frontend**: `useWebSocket` hook maintains an auto-reconnecting connection; `NotificationProvider` shows small toasts and triggers live refresh of the Order Book (depth + open orders) and Balances pages.

## Quick Start

1. **Infrastructure**
   ```bash
   docker-compose up -d
   ```
   Starts Kafka, Zookeeper, Postgres, and MongoDB.

2. **API Gateway**
   ```bash
   cd gateway && go mod tidy && go run ./cmd/gateway
   ```
   Listens on `:8000`. All client traffic goes through here.

3. **API Service**
   ```bash
   cd api && go mod tidy && go run ./cmd/api
   ```
   Listens on `:8080` (internal, behind gateway).

4. **Matching Engine**
   ```bash
   cd matching-engine && go mod tidy && go run ./cmd/matching-engine
   ```
   Restores resting orders from Postgres on boot, then consumes `orders.created` / `orders.cancelled` from Kafka.
   Requires `DB_DSN` env var (defaults to `postgres://dex:dex@localhost:5432/fluxmesh?sslmode=disable`).

5. **Indexer**
   ```bash
   cd indexer && go mod tidy && go run ./cmd/indexer
   ```
   Projects Kafka events into Postgres (order statuses, trades, balances). Health on `:8082`.

6. **Event Log**
   ```bash
   cd eventlog && go mod tidy && go run ./cmd/eventlog
   ```
   Consumes all Kafka topics and writes to MongoDB.

7. **Frontend**
   ```bash
   cd frontend && npm install && npm run dev
   ```
   Vite dev server on `:3000`, proxies `/api` and `/control` to the gateway.

8. **Control plane**
   ```bash
   cd mcp && go mod tidy && go run ./cmd/mcp
   ```

9. **MCP server (Model Context Protocol — for AI assistants)**
   ```bash
   cd mcp && go run ./cmd/fluxmesh-mcp
   ```

10. **Notification WebSocket**
    ```bash
    cd notification && go mod tidy && go run ./cmd/notification
    ```
    Exposes `ws://localhost:8090/ws?token=<jwt>` for real-time user notifications.

## Authentication Flow

1. `POST /auth/register` with `{ email, password }` → creates user (bcrypt-hashed), returns JWT
2. `POST /auth/login` with `{ email, password }` → validates via bcrypt, returns JWT
3. All subsequent requests include `Authorization: Bearer <access_token>`
4. Gateway validates the token and injects identity headers for downstream services
5. React frontend stores the token in `localStorage` and attaches it via `apiFetch()` wrapper

### User Profile

- `GET /profile` — retrieve current user's profile (name, email, avatar)
- `PUT /profile` — update name, email, or avatar URL; publishes `users.updated` to Kafka
- `DELETE /profile` — soft-delete account; publishes `users.deleted` to Kafka

**Dev credentials** (seeded on first startup):
- Trader: `trader@example.com` / `trader123`
- Admin: `admin@example.com` / `admin123`

## Rate Limiting

- **Algorithm**: Token bucket (via `golang.org/x/time/rate`)
- **Per authenticated user**: 20 requests/second, burst of 40
- **Per IP (unauthenticated)**: Same limits, keyed by IP
- **Response on limit exceeded**: `429 Too Many Requests` with `Retry-After: 1` header

## Frontend Routing

| Path | Auth Required | Description |
|------|:---:|-------------|
| `/` | No | Public landing page |
| `/login` | No | Sign-in / Register form |
| `/trade/markets` | Yes | Market list |
| `/trade/markets/:id` | Yes | Order book + place orders |
| `/trade/balances` | Yes | User balances |
| `/trade/profile` | Yes | View/edit profile, delete account |
| `/admin/*` | Yes (admin) | Config, health dashboard |

## Order Cancellation

Orders flow through a strict lifecycle. Cancellation is only allowed on **resting** orders that have not yet been fully executed.

### Cancellation Rules

| Order Status | Cancellable? | Behaviour |
|:-------------|:---:|-----------|
| **Pending** (resting on book) | Yes | Full cancel — remaining quantity removed from book |
| **Partial** (partially filled) | Remaining only | Unfilled portion is cancelled; filled portion is final |
| **Matched** (fully filled) | No | Trade already executed, position taken |
| **Rejected** | No | Order was never on the book |
| **Cancelled** | No | Already cancelled |
| **Market order** | No | Executes instantly at market price — never rests on the book |
| **Post-settlement** | No | On-chain settlement is final |

### Cancellation Fee

Every cancellation incurs a small fee to discourage order-book spam:

```
cancel_fee = remaining_qty × price × market.cancel_fee_rate
```

| Rule | Detail |
|------|--------|
| **Fee rate** | Per-market, stored in `markets.cancel_fee_rate` (default 0.05% / 5 bps) |
| **Fee asset** | Quote asset for buy orders (e.g. USDC), base asset for sell orders (e.g. BTC) |
| **Cap** | Fee is capped at the user's available balance — never goes negative |
| **Deduction** | Fee is deducted atomically from the user's balance inside the stored procedure |
| **Audit** | Fee amount is recorded on the order (`orders.cancel_fee`) and included in the `orders.cancelled` Kafka event |

### Cancel Flow (End-to-End)

```
DELETE /orders/:id  →  API Service  →  fn_cancel_order (Postgres stored proc)
                                            │  Row lock (SELECT FOR UPDATE)
                                            │  Status & type guard
                                            │  Compute & deduct fee
                                            │  Set status = 'cancelled'
                                            ▼
                                    Kafka: orders.cancelled
                                            │
                    ┌───────────────────┬────┴──────────────────┐
                    ▼                   ▼                       ▼
            Matching Engine        Indexer                Event Log
          (remove from book)   (update Postgres)       (MongoDB audit)
```

### HTTP Responses

| Scenario | Status | Body |
|----------|:------:|------|
| Order cancelled | `200 OK` | Cancelled order JSON (includes `cancel_fee`) |
| Order not found | `404 Not Found` | `"order not found"` |
| Non-cancellable state | `409 Conflict` | `"order cannot be cancelled (already filled, rejected, or cancelled)"` |

## Stored Procedures (PostgreSQL)

All database access goes through **PL/pgSQL stored functions** (`CREATE OR REPLACE FUNCTION`), giving us:

- **Atomicity** — multi-step mutations (e.g. cancel + fee deduction) run in a single transaction
- **Row-level locking** — `SELECT ... FOR UPDATE` prevents concurrent modifications
- **Centralized business rules** — status guards and fee logic live in SQL, not scattered in Go code
- **Idempotency** — `CREATE OR REPLACE` makes migrations safe to re-run on every startup

| Function | Purpose |
|----------|---------|
| `fn_create_order_atomic` | Idempotency check + insert in one transaction |
| `fn_cancel_order` | Status/type guard + cancellation fee + balance deduction |
| `fn_process_order_matched` | Atomically update both order statuses + insert trade record |
| `fn_register_user_atomic` | Email uniqueness check + insert (catches race via `EXCEPTION`) |
| `fn_update_profile_atomic` | Row lock + email uniqueness + update |
| `fn_upsert_balance` | `INSERT ... ON CONFLICT DO UPDATE` for balance projections |
| `fn_get_resting_orders` | Returns all `pending`/`partial` orders for matching engine startup recovery |
| 13 more | CRUD for orders, users, markets, balances, trades |

## Matching Engine Persistence & Recovery

The in-memory order book is **rebuilt from Postgres on every startup**, so a restart never loses resting orders.

### Startup Sequence

```
1. Connect to Postgres
2. Call fn_get_resting_orders()  →  all pending/partial orders, sorted by created_at
3. For each order:  engine.RestoreOrder(...)  →  book.Add() (no matching attempted)
4. Close DB connection (engine only uses Kafka at runtime)
5. Begin consuming orders.created + orders.cancelled from Kafka
```

### Why DB Replay (Not Kafka Replay)?

| Approach | Pros | Cons |
|----------|------|------|
| **Postgres query** (chosen) | Single query, instant, source of truth, correct-by-construction | Requires DB access on startup |
| Kafka offset reset | No DB dependency | Must replay entire topic history; filled/cancelled orders must be tracked and skipped; slow on large topics |
| File snapshot | Fast, no external deps | Stale if engine crashed mid-write; two sources of truth |

The Postgres approach is the simplest and most correct: the database already tracks which orders are still resting (`status IN ('pending','partial')`), so one query gives us exactly the set of orders that belong on the book.

### Decimal Precision

Prices and sizes throughout the matching engine and settlement service use `shopspring/decimal` (arbitrary-precision decimals) instead of `float64`, eliminating rounding errors on real trades.

## Resilience & Retry Strategy

Every layer uses **exponential backoff with jitter** to avoid thundering-herd retries.

| Layer | What's retried | Max retries | Base delay | Notes |
|-------|---------------|:-----------:|:----------:|-------|
| **Frontend (`apiFetch`)** | Network errors + 502/503/504 | 3 | 500 ms | GET/HEAD/OPTIONS retried on 5xx; mutations only on network errors (request never reached server) |
| **API Gateway (reverse proxy)** | Connection refused + 502/503/504 | 2 | 150 ms | Network errors retried for all methods; HTTP 5xx only for idempotent methods |
| **Kafka Producer** | Transient broker/network errors | 3 | 200 ms | Respects context cancellation; non-transient errors (serialization) fail immediately |
| **Event Log → MongoDB** | All MongoDB write failures | 4 | 300 ms | Drops event after exhausting retries and logs a warning; commits Kafka offset to avoid reprocessing |

### Idempotency Keys (Duplicate Order Prevention)

Even with safe retry logic, a network drop *after* the server accepts the order but *before* the response reaches the client would cause a retry that creates a duplicate. To handle this:

1. **Frontend** generates a `crypto.randomUUID()` per form submission.
2. Sends it as the `Idempotency-Key` HTTP header on `POST /orders`.
3. **API Service** checks Postgres for an existing order with that key.
4. If found → returns the original order with `200 OK` (no duplicate created).
5. If not found → creates a new order and stores the key (unique index prevents races).

This means the same order submission can be safely retried any number of times and will only ever produce one order in the system.

### Design Decisions

- **Idempotency guard**: POST/PUT/DELETE are *not* retried after a server response (even 5xx) to prevent duplicate side-effects. They *are* retried on network errors because the request never reached the upstream. Additionally, `POST /orders` uses an `Idempotency-Key` header so even retried network-error requests produce at most one order.
- **Jitter**: Every backoff includes random jitter to avoid synchronized retry storms across clients.
- **Fail-fast on auth**: 401 responses are never retried — the frontend immediately clears the token and redirects to login.
- **Circuit breaking**: Not yet implemented. For a production deployment, wrap the gateway proxy and Kafka producer with a circuit breaker (e.g. `sony/gobreaker`) to avoid hammering a degraded upstream.

## Tradeoffs & Design Notes

- **Gateway-first auth**: JWT validation happens once at the gateway edge. Downstream services trust injected headers, eliminating redundant crypto operations and keeping handler latency minimal.
- **Token bucket rate limiting**: Simple, memory-efficient, and fair per-user. For horizontal scaling, swap to Redis-backed distributed rate limiting.
- **MongoDB event log**: Every Kafka event is persisted for audit, debugging, and analytics. Schema-flexible documents handle varying payloads without migrations.
- **Postgres for state, Mongo for events**: Postgres stores the source of truth (users, orders, markets). MongoDB stores the immutable event stream for querying and replay.
- **Event-driven vs synchronous**: Orders are accepted via API and processed asynchronously via Kafka; clients get real-time updates via WebSocket.
- **Why Kafka**: Durable, ordered event log; replay and multiple consumers; aligns with control plane broadcasting.
- **MCP (Model Context Protocol)**: Lets AI tools interact with the DEX without custom integrations.

### Solidity & EVM Integration

- `contracts/ExchangeCore.sol` implements the on-chain settlement core:
  - Maintains per-trade idempotency via `tradeSettled[tradeId]` so off-chain retries are safe.
  - Uses a simple `owner` / `settler` access model plus a non-reentrant `settleTrades` entrypoint.
- `contracts/MarketRegistry.sol` holds market parameters on-chain behind an `owner` gate for config changes.
- The Go `settlement/` service is wired to consume `orders.matched` and emit `trades.settled` / `balances.updated`; connecting it to a live Ethereum RPC + deployed `ExchangeCore` is optional and controlled via environment variables (`ETH_RPC_URL`, `EXCHANGE_CORE_ADDRESS`, `SETTLEMENT_PRIVATE_KEY`, `CHAIN_ID`).
- You do **not** need solc/Hardhat/Foundry installed just to run the backend; Solidity tooling is only required if you want to compile/deploy the contracts yourself.

## Testing

| Layer | Location | How to run |
|-------|----------|------------|
| **Unit (matching engine)** | `matching-engine/internal/orderbook/book_test.go`, `matching-engine/internal/engine/engine_test.go` | `cd matching-engine && go test ./...` |
| **Integration (API)** | `api/cmd/api/integration_test.go` | Requires Postgres + Kafka. `cd api && go test -tags=integration ./cmd/api/...` (skips if DB unreachable) |
| **E2E (order lifecycle)** | Same file as integration | `go test -tags=integration ./cmd/api/... -run TestE2E_OrderLifecycle` — register → login → create order → list → cancel |

- **Matching engine unit tests** cover the order book: add/cancel, match incoming (buy/sell, no fill, full fill, partial fill, two makers), and the engine (reject invalid side/price/size, rest on book, one fill, cancel, restore).
- **API integration tests** hit the real HTTP router (with test DB and Kafka); they check login, register, markets list, depth, auth-required create order, and create order success.
- **E2E** runs a single flow through the API to assert the full order lifecycle with a real DB.

## Interactive API Docs (Swagger)

The gateway serves a **Swagger UI** at `http://localhost:8000/docs` — interactive documentation for every endpoint (auth, profile, orders, markets, balances, admin). The OpenAPI spec is at `docs/swagger.yaml`.

## Docs & Diagrams

- `docs/architecture.md` — System architecture (gateway, services, data stores)
- `docs/kafka-topics.md` — All Kafka topics, consumer groups, offset strategy
- `docs/sequence-order-lifecycle.md` — Order lifecycle sequence diagram
- `docs/sequence-config-lifecycle.md` — Config change lifecycle sequence diagram
- `docs/mcp-model-context-protocol.md` — MCP server and tools for AI
- `docs/swagger.yaml` — OpenAPI 3.0 specification

## License

MIT
