# API Service

REST gateway for the FluxMesh DEX. Handles user authentication, order placement, market discovery, and balance queries. Sits behind the API Gateway (which validates JWTs and injects identity headers), persists state to Postgres via GORM, and publishes domain events to Kafka so downstream services (matching engine, settlement, indexer) can react asynchronously.

## Architecture

```
Client ──▶ API Gateway (JWT verify) ──▶ API Service ──▶ Postgres
                                             │
                                             ├──▶ Kafka  orders.created
                                             ├──▶ Kafka  orders.cancelled
                                             ├──▶ Kafka  users.updated
                                             └──▶ Kafka  users.deleted
```

The gateway strips the `Authorization` header and forwards `X-User-ID` / `X-Role` to the API. `GatewayMiddleware` trusts these headers on protected routes. The `/auth/*` endpoints are public — they issue JWTs directly so the gateway can validate subsequent requests.

## Package Layout

```
api/
├── cmd/api/main.go                  # Entry point, DI wiring, route registration
├── internal/
│   ├── config/config.go             # Env-based configuration loader
│   ├── auth/
│   │   ├── context.go               # Context helpers: WithUser, UserIDFrom, RoleFrom
│   │   ├── gateway.go               # GatewayMiddleware (trusts X-User-ID / X-Role)
│   │   └── jwt.go                   # JWT issue / parse, AuthMiddleware
│   ├── models/
│   │   ├── user.go                  # User entity (UUID PK, bcrypt hash, role)
│   │   ├── order.go                 # Order entity (side, type, status, idempotency key)
│   │   ├── market.go                # Market / trading-pair entity
│   │   └── balance.go               # Per-user per-asset balance (read model)
│   ├── repository/
│   │   ├── user_repository.go       # CRUD for users (interface + GORM impl)
│   │   ├── order_repository.go      # CRUD for orders, idempotency-key lookup
│   │   ├── market_repository.go     # List / get markets
│   │   └── balance_repository.go    # List balances by user
│   ├── service/
│   │   ├── errors.go                # Sentinel errors (ErrMarketDisabled, ErrInvalidSide)
│   │   ├── user_service.go          # Register, Authenticate, profile CRUD, Kafka publish
│   │   ├── order_service.go         # Create/cancel/list orders, Kafka publish
│   │   └── market_service.go        # List/get markets (thin wrapper over repo)
│   ├── kafka/
│   │   └── producer.go              # Kafka writer with retry + circuit breaker (gobreaker)
│   └── dbseed/
│       ├── markets.go               # Seeds default markets (BTC, ETH, SOL, ARB, OP)
│       └── users.go                 # Seeds default dev users (admin + trader)
└── go.mod
```

| Package | Purpose |
|---------|---------|
| `cmd/api` | Bootstrap: DB connect, auto-migrate, seed, wire repos → services → handlers, start HTTP |
| `internal/config` | Reads `DB_DSN`, `KAFKA_BROKERS`, `HTTP_PORT`, `JWT_SECRET` from env with sensible defaults |
| `internal/auth` | JWT creation/parsing, gateway middleware, context user extraction |
| `internal/models` | GORM entities: `User`, `Order`, `Market`, `Balance` |
| `internal/repository` | Database access layer (one repo per model, GORM) |
| `internal/service` | Business logic: validation, bcrypt, idempotency, Kafka publishing |
| `internal/kafka` | `Producer` with per-topic writers and retry with jitter |
| `internal/handler` | HTTP controllers (Chi handlers) — one per domain resource |
| `internal/dbseed` | Dev/local seed data for markets and users |

## Key Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/auth/register` | Public | Create account (email + password + optional role), returns JWT |
| `POST` | `/auth/login` | Public | Authenticate with email + password, returns JWT |
| `GET` | `/profile` | Gateway | Get current user profile |
| `PUT` | `/profile` | Gateway | Update name, email, or avatar; publishes `users.updated` |
| `DELETE` | `/profile` | Gateway | Soft-delete account; publishes `users.deleted` |
| `GET` | `/orders` | Gateway | List user orders (query: `market_id`, `status`) |
| `POST` | `/orders` | Gateway | Place a limit order; supports `Idempotency-Key` header; publishes `orders.created` |
| `DELETE` | `/orders/{id}` | Gateway | Cancel an order (sets status → cancelled); publishes `orders.cancelled` |
| `GET` | `/markets` | Gateway | List all enabled markets |
| `GET` | `/markets/{id}` | Gateway | Get a single market by ID (e.g. `BTC-USDC`) |
| `GET` | `/balances` | Gateway | List per-asset balances for current user |

**Auth note:** "Gateway" means the request must carry `X-User-ID` / `X-Role` headers injected by the API Gateway after JWT validation. Public endpoints validate credentials directly.

## Data Models

### User

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `uuid` | PK, auto-generated |
| `Email` | `string` | Unique, not null |
| `Name` | `string` | Default empty |
| `AvatarURL` | `string` | Default empty |
| `PasswordHash` | `string` | bcrypt, hidden from JSON |
| `Role` | `trader \| admin` | Default `trader` |
| `CreatedAt` | `timestamp` | |
| `UpdatedAt` | `timestamp` | |
| `DeletedAt` | `timestamp` | Soft-delete (GORM) |

### Order

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `uuid` | PK, auto-generated |
| `IdempotencyKey` | `string` | Unique index (max 64 chars) |
| `UserID` | `string` | Indexed, not null |
| `MarketID` | `string` | Indexed, not null |
| `Side` | `buy \| sell` | |
| `Type` | `limit \| market` | |
| `Price` | `numeric` | |
| `Size` | `numeric` | Not null |
| `Remaining` | `numeric` | Unfilled quantity |
| `Status` | `pending \| matched \| partial \| cancelled \| rejected` | Default `pending` |
| `CreatedAt` | `timestamp` | |
| `UpdatedAt` | `timestamp` | |
| `DeletedAt` | `timestamp` | Soft-delete |

### Market

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `string` | PK (e.g. `BTC-USDC`) |
| `BaseAsset` | `string` | Not null |
| `QuoteAsset` | `string` | Not null |
| `TickSize` | `numeric` | Minimum price increment |
| `MinSize` | `numeric` | Minimum order size |
| `FeeRate` | `numeric` | Taker/maker fee rate |
| `Enabled` | `bool` | Default `true` |
| `CreatedAt` | `timestamp` | |
| `UpdatedAt` | `timestamp` | |
| `DeletedAt` | `timestamp` | Soft-delete |

### Balance

| Field | Type | Notes |
|-------|------|-------|
| `UserID` | `string` | Composite PK |
| `Asset` | `string` | Composite PK |
| `Available` | `numeric` | Not null |
| `Locked` | `numeric` | Default `0` |
| `UpdatedAt` | `timestamp` | |
| `DeletedAt` | `timestamp` | Soft-delete |

## Kafka Topics Produced

| Topic | Trigger | Payload Fields |
|-------|---------|----------------|
| `orders.created` | `POST /orders` | `order_id`, `user_id`, `market_id`, `side`, `type`, `price`, `size`, `remaining` |
| `orders.cancelled` | `DELETE /orders/{id}` | `order_id`, `user_id` |
| `users.updated` | `PUT /profile` | `user_id`, `action`, `timestamp`, changed fields (`name`, `old_email`/`new_email`, `avatar_url`) |
| `users.deleted` | `DELETE /profile` | `user_id`, `email`, `action`, `timestamp` |

The producer uses exponential backoff with jitter (base 200ms, max 5s, 3 retries) for transient Kafka errors. It is wrapped with [sony/gobreaker](https://github.com/sony/gobreaker): after 5 consecutive publish failures the circuit opens and publish calls return immediately with `ErrOpenState`; after 30s one probe is allowed. Context cancellation is not counted as a failure so client timeouts do not open the circuit.

## Observability

| Concern | Implementation |
|---------|----------------|
| **Structured logging** | `log/slog` via `internal/logger`; `LOG_FORMAT=json`, `LOG_LEVEL=DEBUG\|INFO\|WARN\|ERROR` |
| **Prometheus** | `GET /metrics` (no auth), `GET /health` (liveness). Metrics: `api_http_requests_total`, `api_http_request_duration_seconds`, `api_kafka_producer_messages_total` (topic, status) |
| **Distributed tracing** | OpenTelemetry with [otelchi](https://github.com/riandyrn/otelchi); extracts trace context from gateway-injected headers and continues the span |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DB_DSN` | `host=localhost user=fluxmesh password=fluxmesh_secret dbname=fluxmesh port=5432 sslmode=disable` | Postgres connection string (GORM DSN) |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker addresses |
| `HTTP_PORT` | `8080` | Port the API listens on |
| `JWT_SECRET` | `change-me-in-production` | HMAC-SHA256 signing key for JWTs |

JWT tokens expire after **60 minutes** (hardcoded in config loader).

## Running

```bash
cd api && go mod tidy && go run ./cmd/api
```
