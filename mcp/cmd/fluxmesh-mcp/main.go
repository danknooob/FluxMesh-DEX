// fluxmesh-mcp runs the Model Context Protocol (MCP) server for FluxMesh DEX.
// AI assistants (e.g. Cursor, Claude) can use tools like get_markets, get_health to query the exchange.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "fluxmesh-dex",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_markets",
		Description: "List enabled trading markets (base/quote, tick size, fee).",
	}, getMarkets)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_health",
		Description: "Return control-plane and data-plane service health status.",
	}, getHealth)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_balances",
		Description: "Get user balances by user_id (placeholder; requires API).",
	}, getBalances)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server: %v", err)
	}
}

type empty struct{}

func textResult(s string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: s}},
	}, nil, nil
}

func getMarkets(ctx context.Context, req *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, any, error) {
	out := map[string]interface{}{
		"markets": []map[string]string{},
		"note":    "Call GET /admin/config or API /markets for live data.",
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return textResult(string(b))
}

func getHealth(ctx context.Context, req *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, any, error) {
	out := map[string]interface{}{
		"services": []map[string]string{
			{"name": "api", "status": "unknown"},
			{"name": "matching-engine", "status": "unknown"},
			{"name": "settlement", "status": "unknown"},
			{"name": "control-plane", "status": "unknown"},
		},
		"note": "Call GET /admin/health for live data.",
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return textResult(string(b))
}

func getBalances(ctx context.Context, req *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, any, error) {
	out := map[string]interface{}{
		"balances": []interface{}{},
		"note":     "Call API GET /balances?user_id=... for live data.",
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return textResult(string(b))
}
