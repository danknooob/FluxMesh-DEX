package service

import (
	"context"
	"log"

	"github.com/danknooob/fluxmesh-dex/api/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
	"github.com/google/uuid"
)

type OrderService struct {
	repo     *repository.OrderRepository
	markets  MarketService
	producer *kafka.Producer
}

func NewOrderService(
	repo *repository.OrderRepository,
	markets MarketService,
	producer *kafka.Producer,
) *OrderService {
	return &OrderService{repo: repo, markets: markets, producer: producer}
}

type CreateLimitOrderRequest struct {
	UserID         string `json:"user_id"`
	MarketID       string `json:"market_id"`
	Side           string `json:"side"`
	Price          string `json:"price"`
	Size           string `json:"size"`
	IdempotencyKey string `json:"-"`
}

// CreateLimitOrder validates, persists, and publishes orders.created.
// The idempotency check + insert is handled atomically by the stored
// function fn_create_order_atomic, eliminating TOCTOU race conditions.
func (s *OrderService) CreateLimitOrder(ctx context.Context, req CreateLimitOrderRequest) (*models.Order, bool, error) {
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

	isDuplicate, err := s.repo.CreateAtomic(ctx, o)
	if err != nil {
		return nil, false, err
	}

	if !isDuplicate {
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
	}

	return o, isDuplicate, nil
}

// CancelOrder cancels a resting order. Only pending and partial orders
// are cancellable — filled, rejected, and already-cancelled orders
// return ErrOrderNotCancellable; missing orders return ErrOrderNotFound.
func (s *OrderService) CancelOrder(ctx context.Context, orderID uuid.UUID, userID string) (*models.Order, error) {
	order, err := s.repo.Delete(ctx, orderID, userID)
	if err != nil {
		if isPgException(err, "ORDER_NOT_FOUND") {
			return nil, ErrOrderNotFound
		}
		if isPgException(err, "ORDER_NOT_CANCELLABLE") {
			return nil, ErrOrderNotCancellable
		}
		return nil, err
	}

	if err := s.producer.PublishOrderCancelled(ctx, map[string]interface{}{
		"order_id":   orderID.String(),
		"user_id":    userID,
		"market_id":  order.MarketID,
		"remaining":  order.Remaining,
		"cancel_fee": order.CancelFee,
	}); err != nil {
		log.Printf("publish orders.cancelled failed: %v", err)
	}

	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context, userID, marketID, status string) ([]models.Order, error) {
	filter := repository.OrderFilter{
		UserID:   userID,
		MarketID: marketID,
		Status:   status,
	}
	return s.repo.List(ctx, filter)
}

func (s *OrderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	return s.repo.GetByID(ctx, id)
}

type DepthResponse struct {
	Bids []repository.PriceLevel `json:"bids"`
	Asks []repository.PriceLevel `json:"asks"`
}

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
