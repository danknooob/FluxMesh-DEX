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
	UserID   string `json:"user_id"`
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Price    string `json:"price"`
	Size     string `json:"size"`
}

// CreateLimitOrder validates, persists, and publishes orders.created.
func (s *OrderService) CreateLimitOrder(ctx context.Context, req CreateLimitOrderRequest) (*models.Order, error) {
	market, err := s.markets.GetMarket(ctx, req.MarketID)
	if err != nil || market == nil {
		return nil, err
	}
	if !market.Enabled {
		return nil, ErrMarketDisabled
	}

	side := models.OrderSide(req.Side)
	if side != models.OrderSideBuy && side != models.OrderSideSell {
		return nil, ErrInvalidSide
	}

	o := &models.Order{
		UserID:    req.UserID,
		MarketID:  req.MarketID,
		Side:      side,
		Type:      models.OrderTypeLimit,
		Price:     req.Price,
		Size:      req.Size,
		Remaining: req.Size,
		Status:    models.OrderStatusPending,
	}
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, err
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
		// Log and still return the order so callers can see the accepted order,
		// even if the event failed to publish.
		log.Printf("publish orders.created failed: %v", err)
	}
	return o, nil
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
