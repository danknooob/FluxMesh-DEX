package handler

import (
	"encoding/json"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
)

// MarketController handles HTTP for markets.
type MarketController struct {
	marketRepo *repository.MarketRepository
}

// NewMarketController creates a MarketController.
func NewMarketController(marketRepo *repository.MarketRepository) *MarketController {
	return &MarketController{marketRepo: marketRepo}
}

// List returns all enabled markets (GET /markets).
func (c *MarketController) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := c.marketRepo.List(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}
