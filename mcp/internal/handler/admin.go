package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"gorm.io/gorm"
)

// AdminHandler serves control-plane admin API (config, health view).
// It depends on Postgres for config (markets) and aggregates health
// from downstream services via simple HTTP probes.
type AdminHandler struct {
	DB *gorm.DB
}

type Market struct {
	ID         string `json:"id"`
	BaseAsset  string `json:"base_asset"`
	QuoteAsset string `json:"quote_asset"`
	TickSize   string `json:"tick_size"`
	MinSize    string `json:"min_size"`
	FeeRate    string `json:"fee_rate"`
	Enabled    bool   `json:"enabled"`
}

// Config returns current config view backed by Postgres.
// For now this is just the markets table; risk/flags remain placeholders.
func (h *AdminHandler) Config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var markets []Market
	if h.DB != nil {
		_ = h.DB.Table("markets").
			Select("id, base_asset, quote_asset, tick_size, COALESCE(min_size, '') AS min_size, fee_rate, enabled").
			Scan(&markets).Error
	}
	out := map[string]interface{}{
		"markets": markets,
		"risk":    map[string]interface{}{},
		"flags":   map[string]bool{},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// Markets is a convenience endpoint used by the admin UI to list markets.
// It simply returns the markets slice from Postgres.
func (h *AdminHandler) Markets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var markets []Market
	if h.DB != nil {
		_ = h.DB.Table("markets").
			Select("id, base_asset, quote_asset, tick_size, COALESCE(min_size, '') AS min_size, fee_rate, enabled").
			Scan(&markets).Error
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(markets)
}

type serviceHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type healthResponse struct {
	Services []serviceHealth `json:"services"`
}

// Health returns aggregated service health. For now it probes the
// indexer /health endpoint and reports others as unknown, so the
// admin UI shows real data instead of static placeholders.
func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	indexerURL := os.Getenv("INDEXER_HEALTH_URL")
	if indexerURL == "" {
		indexerURL = "http://localhost:8082/health"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	status := "unhealthy"
	if resp, err := client.Get(indexerURL); err == nil && resp.StatusCode == http.StatusOK {
		status = "healthy"
	}

	out := healthResponse{
		Services: []serviceHealth{
			{Name: "api", Status: "unknown"},
			{Name: "matching-engine", Status: "unknown"},
			{Name: "settlement", Status: "unknown"},
			{Name: "notification", Status: "unknown"},
			{Name: "indexer", Status: status},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
