package handler

import (
	"encoding/json"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
)

type BalanceController struct {
	repo *repository.BalanceRepository
}

func NewBalanceController(repo *repository.BalanceRepository) *BalanceController {
	return &BalanceController{repo: repo}
}

func (c *BalanceController) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	balances, err := c.repo.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Ensure we never encode nil slice as JSON "null"; return [] instead.
	if balances == nil {
		balances = []models.Balance{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(balances)
}
