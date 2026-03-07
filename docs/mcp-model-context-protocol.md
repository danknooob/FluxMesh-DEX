# MCP — Model Context Protocol

In this project **MCP** means the [Model Context Protocol](https://modelcontextprotocol.io/) (AI tools protocol), not "microservice control plane." The control plane is the HTTP admin service; MCP is how AI assistants talk to the DEX.

## What we expose

The **fluxmesh-mcp** binary runs an MCP server over stdio. AI clients (e.g. Cursor, Claude Desktop) can spawn it and call tools:

| Tool           | Description                                      |
|----------------|--------------------------------------------------|
| `get_markets`  | List enabled trading markets (base/quote, fee).  |
| `get_health`   | Control-plane and data-plane service health.    |
| `get_balances` | User balances (placeholder; use API for live).  |

## Running the MCP server

```bash
cd mcp && go run ./cmd/fluxmesh-mcp
```

The server uses stdin/stdout. Configure your AI client to run this command; it will then have access to the tools above.

## Cursor

In Cursor you can add a custom MCP server that runs `go run ./cmd/fluxmesh-mcp` (or the built binary) from the repo root so the AI can query markets and health.

## Implementation

- **SDK**: `github.com/modelcontextprotocol/go-sdk`
- **Transport**: stdio (one process per client connection).
- **Tools**: Currently return placeholder JSON; in production they can call the control-plane and API HTTP endpoints for live data.
