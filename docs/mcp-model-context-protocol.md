# MCP — Model Context Protocol

In this project **MCP** means the [Model Context Protocol](https://modelcontextprotocol.io/) (AI tools protocol), not "microservice control plane." The control plane is the HTTP admin service; MCP is how AI assistants talk to the DEX.

## What We Expose

The **fluxmesh-mcp** binary runs an MCP server over stdio. AI clients (e.g. Cursor, Claude Desktop) can spawn it and call tools:

| Tool | Description |
|------|-------------|
| `get_markets` | List enabled trading markets (base/quote, tick size, fee rate) |
| `get_health` | Control-plane and data-plane service health status |
| `get_balances` | User balances (requires `user_id` parameter) |

## Running the MCP Server

```bash
cd mcp && go run ./cmd/fluxmesh-mcp
```

The server uses stdin/stdout. Configure your AI client to run this command; it will then have access to the tools above.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_BASE_URL` | `http://localhost:8080` | API service endpoint |
| `CONTROL_BASE_URL` | `http://localhost:8081` | Control plane endpoint |

## Cursor Integration

This repo includes a project-level MCP config so Cursor (and Cloud Agents) can use the FluxMesh DEX tools without extra setup.

- **Config file:** [`.cursor/mcp.json`](../.cursor/mcp.json) — defines the `fluxmesh-dex` MCP server (runs `go run ./cmd/fluxmesh-mcp` from the `mcp/` directory).
- **Restart Cursor** after changing MCP config for it to take effect.
- Ensure the API and Control Plane are reachable at `API_BASE_URL` and `CONTROL_BASE_URL` (defaults: `http://localhost:8080`, `http://localhost:8081`) when using the tools.

## Implementation

- **SDK**: `github.com/modelcontextprotocol/go-sdk`
- **Transport**: stdio (one process per client connection)
- **Data source**: Tools call the API and control plane HTTP endpoints for live data
