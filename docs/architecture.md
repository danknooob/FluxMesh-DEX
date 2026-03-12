# System Architecture

## Overview

```
                        ┌───────────────────┐
                        │   React Frontend  │
                        │  (Trader + Admin) │
                        └─────────┬─────────┘
                                  │
                        ┌─────────▼─────────┐
                        │   API Gateway     │  :8000
                        │  ┌─────────────┐  │
                        │  │ JWT Auth     │  │
                        │  │ Rate Limiter │  │
                        │  └─────────────┘  │
                        └───┬───────────┬───┘
                            │           │
               ┌────────────▼──┐   ┌────▼──────────────┐
               │  API Service  │   │  Control Plane     │
               │  :8080        │   │  :8081             │
               │  Auth/Orders  │   │  Config · Health   │
               │  Markets/Bal  │   │  Audit · Commands  │
               │  Postgres     │   └────────────────────┘
               └───────┬───────┘
                       │  Kafka
        ┌──────────────┼──────────────┬──────────────┐
        ▼              ▼              ▼              ▼
  ┌───────────┐ ┌───────────┐ ┌────────────┐ ┌────────────┐
  │ Matching  │ │Settlement │ │Notification│ │ Event Log  │
  │ Engine    │ │ Service   │ │ WebSocket  │ │  → MongoDB │
  └───────────┘ └───────────┘ └────────────┘ └────────────┘
```

## Layers

### Edge — API Gateway (:8000)

The gateway is the single entry point for all client traffic.

- **JWT authentication**: Validates `Authorization: Bearer <token>` on every protected route. Admin routes additionally require `role=admin`.
- **Rate limiting**: Per-user token-bucket (20 req/s, burst 40) via `golang.org/x/time/rate`. Falls back to per-IP for unauthenticated endpoints.
- **Header injection**: After validation, injects `X-User-ID` and `X-Role` headers so downstream services skip token parsing.
- **Reverse proxy**: Routes to API Service (:8080) for trader routes, Control Plane (:8081) for admin routes.

### Data Plane (Runtime Path)

Services that move orders and trades:

1. **API Service** — HTTP gateway. Authenticates users (bcrypt + JWT), manages user profiles, persists orders in Postgres, publishes events (`orders.created`, `users.updated`, `users.deleted`) to Kafka. Trusts gateway-injected headers for user identity.
2. **Matching Engine** — Consumes `orders.created`, maintains in-memory order books (price-time priority), emits `orders.matched` / `orders.rejected`.
3. **Settlement** — Consumes `orders.matched`, batches and calls EVM `ExchangeCore.settleTrades`, emits `trades.settled` and `balances.updated`.
4. **Indexer** — Listens to chain events and `trades.settled`; updates Postgres read models (positions, balances, trade history).
5. **Notification** — Consumes domain topics + `notifications.user`; holds WebSocket connections per user and pushes real-time updates.
6. **Event Log** — Consumes all 13 Kafka topics and persists every event to MongoDB with a human-readable title. Serves as immutable audit trail.

### Control Plane

Manages the system:

- **Configuration & desired state** — Markets, tick size, fees, risk limits, feature flags. Stored in control-plane DB; changes published to `control.config`.
- **Service registry and health** — Services heartbeat via `control.health`. Control plane aggregates and exposes in admin UI.
- **Access and audit** — All admin changes logged to `control.audit`.
- **Operational commands** — e.g. "pause market ETH/USDC". Published to `control.commands`; data-plane services subscribe and apply.

### MCP (Model Context Protocol)

A separate MCP server exposes DEX capabilities to AI assistants (Cursor, Claude): tools like `get_markets`, `get_balances`, `get_health`.

## Data Stores

| Store | Purpose | Data |
|-------|---------|------|
| **Postgres** | Source of truth | Users (bcrypt hashes, profiles), orders, markets, balances |
| **MongoDB** | Immutable event log | Every Kafka event with topic, human-readable title, payload, timestamps |
| **Kafka** | Event bus | Async communication between all services |
| **In-memory** | Hot data | Order books in matching engine, rate limit buckets in gateway |

## Authentication Flow

```
Client → POST /auth/register or /auth/login → API Service → Postgres (bcrypt)
                                             ← JWT (HS256, 60min expiry)

Client → GET /orders (Bearer token) → Gateway (validate JWT, rate limit)
                                     → inject X-User-ID, X-Role headers
                                     → API Service (trust headers, execute logic)
```

## User Profile Management

```
GET  /profile   → Gateway → API Service → Postgres → user profile JSON
PUT  /profile   → Gateway → API Service → Postgres update + Kafka(users.updated)
DELETE /profile → Gateway → API Service → Postgres soft-delete + Kafka(users.deleted)
```

Profile changes publish events to Kafka, which the Event Log service persists to MongoDB with titles like "Profile updated: alice@example.com changed name" or "Account deleted: user-uuid".

## Request Lifecycle

```
Client → Gateway → JWT check → Rate limit → Reverse proxy → API Service → Postgres/Kafka
                                                           ← JSON response
         ← JSON response (or 401/429)
```

## Interactive API Docs

The API Gateway serves **Swagger UI** at `GET /docs` (port 8000). The OpenAPI spec lives at `docs/swagger.yaml` and documents every endpoint including auth, profile, orders, markets, balances, and admin routes.
