package handler

import (
	"encoding/json"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// OrderController handles HTTP for orders.
type OrderController struct {
	orderService *service.OrderService
}

// NewOrderController creates an OrderController.
func NewOrderController(orderService *service.OrderService) *OrderController {
	return &OrderController{orderService: orderService}
}

// List returns orders for the current user (GET /orders).
// Filters: market_id, status via query params.
func (c *OrderController) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// TODO: derive userID from auth/JWT
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default-user"
	}
	marketID := r.URL.Query().Get("market_id")
	status := r.URL.Query().Get("status")

	orders, err := c.orderService.ListOrders(r.Context(), userID, marketID, status)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(orders)
}

// Create parses JSON body and creates a limit order (POST /orders).
func (c *OrderController) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req service.CreateLimitOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	// TODO: set req.UserID from JWT or wallet auth
	if req.UserID == "" {
		req.UserID = "default-user"
	}

	order, err := c.orderService.CreateLimitOrder(r.Context(), req)
	if err != nil {
		if err == service.ErrMarketDisabled || err == service.ErrInvalidSide {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(order)
}

// Delete cancels an order (DELETE /orders/:id).
func (c *OrderController) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}
	userID := "default-user" // TODO: from auth

	if err := c.orderService.CancelOrder(r.Context(), id, userID); err != nil {
		http.Error(w, "cancel failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
