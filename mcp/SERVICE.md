# MCP Service

Contains **two** servers in a single Go module: a **Control Plane** admin HTTP API for exchange ops, and a **Model Context Protocol (MCP)** AI-tools server that exposes FluxMesh DEX to AI assistants like Cursor and Claude.

## Purpose

This module serves a dual role:

1. **Control Plane** (`cmd/mcp/main.go`) — An HTTP server that exposes admin endpoints for viewing exchange configuration and aggregated service health. It is the central ops dashboard back-end.
2. **MCP AI-Tools Server** (`cmd/fluxmesh-mcp/main.go`) — An MCP-compliant stdio server that registers tools (`get_markets`, `get_health`, `get_balances`) so AI assistants can query the exchange through the Model Context Protocol.

```
                         ┌──────────────────────────────┐
                         │         mcp/ module           │
                         │                              │
  Admin / Ops ──HTTP──▶  │  cmd/mcp          (port 8081)│
                         │    /admin/config              │
                         │    /admin/health              │
                         │                              │
  AI Assistant ─stdio─▶  │  cmd/fluxmesh-mcp (stdio)   │
                         │    get_markets   → API       │
                         │    get_health    → Control   │
                         │    get_balances  → API       │
                         └──────────────────────────────┘
                                │              │
                                ▼              ▼
                          API service    Control Plane
                         (port 8080)     (port 8081)
```

## Entry Points

### 1. Control Plane — `cmd/mcp/main.go`

Standard `net/http` server that loads config from environment and registers admin routes on an `http.ServeMux`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/config` | Returns current exchange configuration (markets, risk params, feature flags) |
| GET | `/admin/health` | Returns aggregated health status of all services |

Both endpoints currently return **hardcoded stubs** (empty market list, all services `"unknown"`).

### 2. MCP AI-Tools Server — `cmd/fluxmesh-mcp/main.go`

Runs over **stdio** using the `modelcontextprotocol/go-sdk`. AI agents call registered tools via the MCP protocol.

| Tool | Description | Upstream Call |
|------|-------------|---------------|
| `get_markets` | List enabled trading markets (base/quote, tick size, fee) | `GET {API_BASE_URL}/markets` |
| `get_health` | Service health status across data and control planes | `GET {CONTROL_BASE_URL}/admin/health` |
| `get_balances` | Fetch balances for a user by `user_id` | `GET {API_BASE_URL}/balances?user_id=<id>` |

Each tool is a thin HTTP proxy — it calls the API or Control Plane, parses the JSON response, and returns it as MCP `TextContent`.

## Kafka Topics (Constants)

The `internal/kafka` package defines control-plane topic names used across the module:

| Constant | Topic | Purpose |
|----------|-------|---------|
| `TopicConfig` | `control.config` | Configuration change events |
| `TopicHealth` | `control.health` | Service health heartbeats |
| `TopicAudit` | `control.audit` | Admin audit trail events |
| `TopicCommands` | `control.commands` | Operational commands (restart, pause, etc.) |

> These constants are defined but **not yet consumed or produced** by either entry point.

## Package Layout

```
mcp/
├── cmd/
│   ├── mcp/
│   │   └── main.go                   # Control Plane HTTP server (admin API)
│   └── fluxmesh-mcp/
│       └── main.go                   # MCP AI-tools stdio server
├── internal/
│   ├── config/
│   │   └── config.go                 # Config struct, env loading (HTTP_PORT, DB_DSN, KAFKA_BROKERS)
│   ├── handler/
│   │   └── admin.go                  # AdminHandler — /admin/config, /admin/health (stubs)
│   └── kafka/
│       └── topics.go                 # Control-plane topic name constants
└── go.mod
```

| Package | Responsibility |
|---------|---------------|
| `cmd/mcp` | Wire config → admin handler → HTTP mux → `ListenAndServe` |
| `cmd/fluxmesh-mcp` | Register MCP tools, create stdio transport, run MCP server loop |
| `internal/config` | Load `HTTP_PORT`, `DB_DSN`, `KAFKA_BROKERS` from environment |
| `internal/handler` | Admin endpoint handlers (currently hardcoded stubs) |
| `internal/kafka` | Topic name constants for control-plane Kafka topics |

## Configuration

### Control Plane (`cmd/mcp`)

| Env Var | Default | Description |
|---------|---------|-------------|
| `HTTP_PORT` | `8081` | HTTP listen port for admin API |
| `DB_DSN` | `host=localhost user=fluxmesh password=fluxmesh_secret dbname=fluxmesh port=5432 sslmode=disable` | Postgres DSN (not yet used) |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka brokers (not yet used) |

### MCP AI-Tools Server (`cmd/fluxmesh-mcp`)

| Env Var | Default | Description |
|---------|---------|-------------|
| `API_BASE_URL` | `http://localhost:8080` | Base URL of the FluxMesh API service |
| `CONTROL_BASE_URL` | `http://localhost:8081` | Base URL of the Control Plane (this module's HTTP server) |

## Running

### Control Plane

```bash
cd mcp && go mod tidy && go run ./cmd/mcp
```

The admin API will be available at `http://localhost:8081`.

### MCP AI-Tools Server

```bash
cd mcp && go mod tidy && go run ./cmd/fluxmesh-mcp
```

This starts a stdio-based MCP server. To use it with Cursor, add it to your MCP config:

```json
{
  "mcpServers": {
    "fluxmesh-dex": {
      "command": "go",
      "args": ["run", "./cmd/fluxmesh-mcp"],
      "cwd": "mcp/"
    }
  }
}
```

## Known Limitations

- **Hardcoded stubs** — `/admin/config` returns an empty market list and `/admin/health` returns `"unknown"` for all services. No database or Kafka integration is wired yet.
- **No authentication** — admin endpoints are unprotected; production requires API key or mTLS.
- **Kafka topics unused** — `control.*` topic constants are defined but neither consumed nor produced.
- **DB_DSN unused** — Postgres DSN is loaded into config but never opened.
- **MCP tools are read-only proxies** — no write/mutation tools (e.g. place order, pause market) are exposed yet.
