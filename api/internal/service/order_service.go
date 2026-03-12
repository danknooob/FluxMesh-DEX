package service

import (
	"context"
	"log"

	"github.com/danknooob/fluxmesh-dex/api/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
	"github.com/google/uuid"
)

// OrderService implements order creation and cancellation.
type OrderService struct {
	repo       *repository.OrderRepository
	markets    MarketService
	producer   *kafka.Producer
}

// NewOrderService creates an OrderService.
func NewOrderService(
	repo *repository.OrderRepository,
	markets MarketService,
	producer *kafka.Producer,
) *OrderService {
	return &OrderService{repo: repo, markets: markets, producer: producer}
}

// CreateLimitOrderRequest is the input for creating a limit order.
type CreateLimitOrderRequest struct {
	UserID         string `json:"user_id"`
	MarketID       string `json:"market_id"`
	Side           string `json:"side"`
	Price          string `json:"price"`
	Size           string `json:"size"`
	IdempotencyKey string `json:"-"`
}

// CreateLimitOrder validates, persists, and publishes orders.created.
// If an IdempotencyKey is provided and an order with that key already exists,
// the original order is returned without creating a duplicate.
func (s *OrderService) CreateLimitOrder(ctx context.Context, req CreateLimitOrderRequest) (*models.Order, bool, error) {
	if req.IdempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, false, err
		}
		if existing != nil {
			return existing, true, nil
		}
	}

	market, err := s.markets.GetMarket(ctx, req.MarketID)
	if err != nil || market == nil {
		return nil, false, err
	}
	if !market.Enabled {
		return nil, false, ErrMarketDisabled
	}

	side := models.OrderSide(req.Side)
	if side != models.OrderSideBuy && side != models.OrderSideSell {
		return nil, false, ErrInvalidSide
	}

	o := &models.Order{
		IdempotencyKey: req.IdempotencyKey,
		UserID:         req.UserID,
		MarketID:       req.MarketID,
		Side:           side,
		Type:           models.OrderTypeLimit,
		Price:          req.Price,
		Size:           req.Size,
		Remaining:      req.Size,
		Status:         models.OrderStatusPending,
	}
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, false, err
	}

	event := map[string]interface{}{
		"order_id":  o.ID.String(),
		"user_id":   o.UserID,
		"market_id": o.MarketID,
		"side":      string(o.Side),
		"type":      string(o.Type),
		"price":     o.Price,
		"size":      o.Size,
		"remaining": o.Remaining,
	}
	if err := s.producer.PublishOrderCreated(ctx, event); err != nil {
		log.Printf("publish orders.created failed: %v", err)
	}
	return o, false, nil
}

// CancelOrder cancels an order and publishes orders.cancelled.
func (s *OrderService) CancelOrder(ctx context.Context, orderID uuid.UUID, userID string) error {
	if err := s.repo.Delete(ctx, orderID, userID); err != nil {
		return err
	}
	return s.producer.PublishOrderCancelled(ctx, map[string]interface{}{
		"order_id": orderID.String(),
		"user_id":  userID,
	})
}

// ListOrders returns orders for a user with optional filters.
// Later, different market types (spot, perps, etc.) can project this
// base order stream into specialized views without changing the interface.
func (s *OrderService) ListOrders(ctx context.Context, userID, marketID, status string) ([]models.Order, error) {
	filter := repository.OrderFilter{
		UserID:   userID,
		MarketID: marketID,
		Status:   status,
	}
	return s.repo.List(ctx, filter)
}

// GetOrder returns a single order by ID.
func (s *OrderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	return s.repo.GetByID(ctx, id)
}

// DepthResponse is the aggregated order book for a market.
type DepthResponse struct {
	Bids []repository.PriceLevel `json:"bids"`
	Asks []repository.PriceLevel `json:"asks"`
}

// GetDepth returns aggregated bids and asks for a market.
func (s *OrderService) GetDepth(ctx context.Context, marketID string, limit int) (*DepthResponse, error) {
	bids, err := s.repo.Depth(ctx, marketID, "buy", limit)
	if err != nil {
		return nil, err
	}
	asks, err := s.repo.Depth(ctx, marketID, "sell", limit)
	if err != nil {
		return nil, err
	}
	return &DepthResponse{Bids: bids, Asks: asks}, nil
}
