package handler

import (
	"encoding/json"
	"net/http"
)

// AdminHandler serves MCP admin API (config, health view).
type AdminHandler struct{}

// Config returns current config view (placeholder).
func (h *AdminHandler) Config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// TODO: load from DB and merge with control.config
	out := map[string]interface{}{
		"markets": []interface{}{},
		"risk":    map[string]interface{}{},
		"flags":   map[string]bool{},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// Health returns aggregated service health (placeholder).
func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// TODO: aggregate from control.health or gRPC/HTTP heartbeats
	out := map[string]interface{}{
		"services": []map[string]interface{}{
			{"name": "api", "status": "unknown"},
			{"name": "matching-engine", "status": "unknown"},
			{"name": "settlement", "status": "unknown"},
			{"name": "notification", "status": "unknown"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
