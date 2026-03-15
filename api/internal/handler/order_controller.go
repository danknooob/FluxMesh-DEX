package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
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
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
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
// Supports idempotency via the Idempotency-Key header — if a duplicate
// key is received, the original order is returned with 200 instead of 202.
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
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	req.UserID = userID
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")

	order, duplicate, err := c.orderService.CreateLimitOrder(r.Context(), req)
	if err != nil {
		switch {
		case err == service.ErrMarketNotFound:
			http.Error(w, "market not found", http.StatusNotFound)
			return
		case err == service.ErrMarketDisabled || err == service.ErrInvalidSide:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			log.Printf("[order] CreateLimitOrder failed: %v", err)
			msg := err.Error()
			switch {
			case strings.Contains(strings.ToLower(msg), "unique") || strings.Contains(strings.ToLower(msg), "duplicate key"):
				http.Error(w, "duplicate order (idempotency key already used)", http.StatusConflict)
				return
			case strings.Contains(strings.ToLower(msg), "foreign key") || strings.Contains(strings.ToLower(msg), "violates foreign key"):
				http.Error(w, "invalid user or market", http.StatusBadRequest)
				return
			case strings.Contains(strings.ToLower(msg), "scan") || strings.Contains(strings.ToLower(msg), "convert"):
				http.Error(w, "order data type error: "+msg, http.StatusInternalServerError)
				return
			default:
				http.Error(w, "internal error: "+msg, http.StatusInternalServerError)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if duplicate {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
	_ = json.NewEncoder(w).Encode(order)
}

// Depth returns aggregated bids/asks for a market (GET /markets/{id}/depth).
func (c *OrderController) Depth(w http.ResponseWriter, r *http.Request) {
	marketID := chi.URLParam(r, "id")
	if marketID == "" {
		http.Error(w, "missing market id", http.StatusBadRequest)
		return
	}
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	depth, err := c.orderService.GetDepth(r.Context(), marketID, limit)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(depth)
}

// Delete cancels a resting order (DELETE /orders/:id).
// Returns 200 + cancelled order on success, 404 if not found,
// 409 if the order is in a non-cancellable state (filled, rejected, etc.).
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
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	order, err := c.orderService.CancelOrder(r.Context(), id, userID)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			http.Error(w, "order not found", http.StatusNotFound)
		case service.ErrOrderNotCancellable:
			http.Error(w, "order cannot be cancelled (already filled, rejected, or cancelled)", http.StatusConflict)
		default:
			http.Error(w, "cancel failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(order)
}
