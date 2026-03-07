package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danknooob/fluxmesh-dex/api/internal/service"
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
	idStr := strings.TrimPrefix(r.URL.Path, "/orders/")
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
