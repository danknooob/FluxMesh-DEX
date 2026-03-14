# API Gateway

Single entry-point reverse proxy that sits in front of every FluxMesh DEX backend. It terminates JWT authentication, enforces per-client rate limits, injects identity headers, and forwards traffic to the appropriate upstream service — so individual backends never deal with auth or throttling.

## Architecture

```
                          ┌──────────────────────────────┐
                          │        API Gateway (:8000)    │
  Clients ───────────────▶│                               │
  (browser / SDK / curl)  │  chi router                   │
                          │    ├─ StripSlashes             │
                          │    ├─ RealIP                   │
                          │    ├─ Logger                   │
                          │    └─ Recoverer                │
                          │                               │
                          │  /docs, /docs/swagger.yaml    │──▶  Swagger UI (static)
                          │  /auth/*          (public)    │──┐
                          │  /profile, /markets, ...      │──┤
                          │  /orders, /balances            │──┤
                          │                               │  │
                          │  /admin/*         (admin JWT) │──┼──┐
                          └──────────────────────────────┘  │  │
                                                            ▼  ▼
                                              ┌──────────┐  ┌──────────────┐
                                              │ API Svc  │  │ Control Svc  │
                                              │ (:8080)  │  │ (:8081)      │
                                              └──────────┘  └──────────────┘
```

## Route Table

| Method | Path | Auth | Rate Limited | Backend |
|--------|------|------|--------------|---------|
| `GET` | `/metrics` | No | No | Prometheus scrape |
| `GET` | `/docs` | No | No | Swagger UI (in-process) |
| `GET` | `/docs/swagger.yaml` | No | No | Swagger spec file |
| `POST` | `/auth/login` | No | No | API service |
| `POST` | `/auth/register` | No | No | API service |
| `GET` | `/profile` | JWT | Yes | API service |
| `PUT` | `/profile` | JWT | Yes | API service |
| `DELETE` | `/profile` | JWT | Yes | API service |
| `GET` | `/markets` | JWT | Yes | API service |
| `GET` | `/markets/{id}` | JWT | Yes | API service |
| `GET` | `/orders` | JWT | Yes | API service |
| `POST` | `/orders` | JWT | Yes | API service |
| `DELETE` | `/orders/{id}` | JWT | Yes | API service |
| `GET` | `/balances` | JWT | Yes | API service |
| `GET/POST/PUT/DELETE` | `/admin/*` | JWT (admin) | Yes | Control service |

## Middleware Pipeline

Requests flow through middleware in order; each layer can short-circuit the chain.

```
Request
  │
  ▼
StripSlashes          chi built-in — normalises trailing slashes
  │
  ▼
RealIP                chi built-in — trusts X-Forwarded-For / X-Real-IP
  │
  ▼
Logger                chi built-in — structured access log
  │
  ▼
Recoverer             chi built-in — catches panics, returns 500
  │
  ▼
JWTAuth               custom — validates Bearer token, injects headers
  │                      ├── 401 if token missing or invalid
  │                      ├── 403 if admin-only route and role ≠ admin
  │                      └── sets X-User-ID, X-Role on proxied request
  ▼
RateLimiter           custom — per-client token bucket
  │                      └── 429 + Retry-After: 1 if exhausted
  ▼
Reverse Proxy         httputil.ReverseProxy with retry transport + circuit breaker
```

The proxy transport is wrapped with [sony/gobreaker](https://github.com/sony/gobreaker). After **5 consecutive upstream failures** (network or 5xx), the circuit opens and the gateway **fails fast** (returns error immediately without calling the backend) for that target. After **30 seconds** the circuit moves to half-open and one probe request is allowed; success closes the circuit, failure reopens it. This prevents sustained upstream outages from hammering the API or control service.

### JWT Auth Details

- Parses `Authorization: Bearer <token>` header.
- Validates signature against `JWT_SECRET` using `golang-jwt/jwt/v5`.
- Extracts `sub` (user ID) and `role` claims.
- Injects `X-User-ID` and `X-Role` headers into the upstream request so backends can trust identity without re-validating.
- Stores `user_id` and `role` in request context for downstream middleware (rate limiter keys off `user_id`).

## Rate Limiting

| Property | Value |
|----------|-------|
| Algorithm | Token bucket (`golang.org/x/time/rate`) |
| Rate | 20 requests / second per client |
| Burst | 40 |
| Client key (authenticated) | `user:<user_id>` from JWT claims |
| Client key (unauthenticated) | `ip:<remote_addr>` |
| Exceeded response | `429 Too Many Requests` with `Retry-After: 1` header |

Limiters are created lazily on first request and stored in a `sync.Mutex`-guarded map.

## Retry Strategy & Circuit Breaker

The reverse proxy wraps `http.DefaultTransport` in a custom `retryTransport`, which is then wrapped in a **circuit breaker** (sony/gobreaker).

**Retry (inside closed circuit):**

| Property | Value |
|----------|-------|
| Max retries | 2 (3 total attempts) |
| Base delay | 150 ms |
| Max delay | 2 s |
| Backoff formula | `150ms × 2^attempt + random jitter (0–150ms)` |
| Network errors (conn refused, timeout) | Retried for **all** HTTP methods |
| HTTP 502 / 503 / 504 | Retried for **idempotent** methods only (`GET`, `HEAD`, `OPTIONS`) |
| Non-idempotent + HTTP error | **Not** retried (request may have reached upstream) |

Request bodies are buffered into memory before the first attempt so they can be replayed on retry.

**Circuit breaker (per upstream):**

| Property | Value |
|----------|-------|
| Open condition | 5 consecutive failures (any error from RoundTrip) |
| Open duration | 30 s, then transition to half-open |
| Half-open | 1 probe request allowed; success → closed, failure → open again |
| When open | No upstream calls; RoundTrip returns error immediately (gateway responds with 502) |
| Logging | State changes (closed → open, open → half-open, etc.) are logged |

## Observability

| Concern | Implementation |
|---------|----------------|
| **Structured logging** | `log/slog` via `internal/logger`; `LOG_FORMAT=json` for JSON, `LOG_LEVEL=DEBUG\|INFO\|WARN\|ERROR` |
| **Prometheus** | `GET /metrics` (no auth). Metrics: `gateway_http_requests_total`, `gateway_http_request_duration_seconds`, `gateway_circuit_breaker_open` |
| **Distributed tracing** | OpenTelemetry with [otelchi](https://github.com/riandyrn/otelchi); trace context is injected into proxied requests (W3C Trace Context) so the API continues the same trace |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `GATEWAY_PORT` | `8000` | Port the gateway listens on |
| `JWT_SECRET` | `change-me-in-production` | HMAC secret for JWT validation |
| `API_TARGET` | `http://localhost:8080` | Upstream URL for the API service |
| `CONTROL_TARGET` | `http://localhost:8081` | Upstream URL for the Control service |
| `SWAGGER_SPEC` | `../docs/swagger.yaml` | Path to the OpenAPI spec file |

## Running

```bash
cd gateway && go mod tidy && go run ./cmd/gateway
```
