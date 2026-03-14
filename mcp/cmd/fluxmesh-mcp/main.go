// fluxmesh-mcp runs the Model Context Protocol (MCP) server for FluxMesh DEX.
// AI assistants (e.g. Cursor, Claude) can use tools like get_markets, get_health to query the exchange.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	apiBase := os.Getenv("API_BASE_URL")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	controlBase := os.Getenv("CONTROL_BASE_URL")
	if controlBase == "" {
		controlBase = "http://localhost:8081"
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "fluxmesh-dex",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_markets",
		Description: "List enabled trading markets (base/quote, tick size, fee).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, any, error) {
		url := apiBase + "/markets"
		body, err := httpGetJSON(ctx, httpClient, url)
		if err != nil {
			return textResultError("failed to fetch markets: " + err.Error())
		}
		return textResult(body)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_health",
		Description: "Return control-plane and data-plane service health status.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, any, error) {
		url := controlBase + "/admin/health"
		body, err := httpGetJSON(ctx, httpClient, url)
		if err != nil {
			return textResultError("failed to fetch health: " + err.Error())
		}
		return textResult(body)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_balances",
		Description: "Get user balances by user_id via the API.",
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

func textResultError(msg string) (*mcp.CallToolResult, any, error) {
	out := map[string]interface{}{
		"error": msg,
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return textResult(string(b))
}

type balancesInput struct {
	UserID string `json:"user_id" jsonschema:"user id whose balances to fetch"`
}

// getBalances uses the API /balances endpoint to fetch balances for a user.
func getBalances(ctx context.Context, req *mcp.CallToolRequest, in balancesInput) (*mcp.CallToolResult, any, error) {
	apiBase := os.Getenv("API_BASE_URL")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}

	if in.UserID == "" {
		return textResultError("user_id is required")
	}

	url := apiBase + "/balances?user_id=" + in.UserID
	body, err := httpGetJSON(ctx, httpClient, url)
	if err != nil {
		return textResultError("failed to fetch balances: " + err.Error())
	}
	return textResult(body)
}

// httpGetJSON performs a simple GET and returns the raw response body as a string.
func httpGetJSON(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var anyJSON interface{}
	if err := json.NewDecoder(resp.Body).Decode(&anyJSON); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(anyJSON, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
