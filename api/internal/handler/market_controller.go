package handler

import (
	"encoding/json"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/service"
	"github.com/go-chi/chi/v5"
)

// MarketController handles HTTP for markets.
type MarketController struct {
	markets service.MarketService
}

// NewMarketController creates a MarketController.
func NewMarketController(markets service.MarketService) *MarketController {
	return &MarketController{markets: markets}
}

// List returns all enabled markets (GET /markets).
func (c *MarketController) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := c.markets.ListMarkets(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

// Get returns a single market by id (GET /markets/{id}).
func (c *MarketController) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing market id", http.StatusBadRequest)
		return
	}
	m, err := c.markets.GetMarket(r.Context(), id)
	if err != nil || m == nil {
		http.Error(w, "market not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}
