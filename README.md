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
| `gateway/` | API Gateway — JWT validation, per-user token-bucket rate limiting, reverse proxy |
| `contracts/` | EVM smart contracts (ExchangeCore, MarketRegistry) |
| `api/` | Go MVC HTTP service (auth, profile, orders, markets, balances, Kafka producer) |
| `matching-engine/` | Order-book matching; consumes `orders.created`, emits `orders.matched` / `orders.rejected` |
| `settlement/` | Consumes `orders.matched`, batches and calls EVM `ExchangeCore.settleTrades` |
| `notification/` | WebSocket service; consumes domain + notification topics |
| `eventlog/` | Kafka → MongoDB event logger; subscribes to all topics and persists every event |
| `mcp/` | Control plane HTTP API + MCP (Model Context Protocol) server with DEX tools for AI |
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

4. **Event Log**
   ```bash
   cd eventlog && go mod tidy && go run ./cmd/eventlog
   ```
   Consumes all Kafka topics and writes to MongoDB.

5. **Frontend**
   ```bash
   cd frontend && npm install && npm run dev
   ```
   Vite dev server on `:3000`, proxies `/api` and `/control` to the gateway.

6. **Control plane**
   ```bash
   cd mcp && go mod tidy && go run ./cmd/mcp
   ```

7. **MCP server (Model Context Protocol — for AI assistants)**
   ```bash
   cd mcp && go run ./cmd/fluxmesh-mcp
   ```

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

## Tradeoffs & Design Notes

- **Gateway-first auth**: JWT validation happens once at the gateway edge. Downstream services trust injected headers, eliminating redundant crypto operations and keeping handler latency minimal.
- **Token bucket rate limiting**: Simple, memory-efficient, and fair per-user. For horizontal scaling, swap to Redis-backed distributed rate limiting.
- **MongoDB event log**: Every Kafka event is persisted for audit, debugging, and analytics. Schema-flexible documents handle varying payloads without migrations.
- **Postgres for state, Mongo for events**: Postgres stores the source of truth (users, orders, markets). MongoDB stores the immutable event stream for querying and replay.
- **Event-driven vs synchronous**: Orders are accepted via API and processed asynchronously via Kafka; clients get real-time updates via WebSocket.
- **Why Kafka**: Durable, ordered event log; replay and multiple consumers; aligns with control plane broadcasting.
- **MCP (Model Context Protocol)**: Lets AI tools interact with the DEX without custom integrations.

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
