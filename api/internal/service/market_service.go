package service

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
)

// MarketService defines the interface for working with markets.
// All features that need market data should depend on this interface
// rather than reaching directly into the repository or database.
type MarketService interface {
	ListMarkets(ctx context.Context) ([]models.Market, error)
	GetMarket(ctx context.Context, id string) (*models.Market, error)
}

type marketService struct {
	repo *repository.MarketRepository
}

// NewMarketService creates a MarketService backed by a MarketRepository.
func NewMarketService(repo *repository.MarketRepository) MarketService {
	return &marketService{repo: repo}
}

// ListMarkets returns all enabled markets.
func (s *marketService) ListMarkets(ctx context.Context) ([]models.Market, error) {
	return s.repo.List(ctx)
}

// GetMarket returns a single market by its id (e.g. "BTC-USDC").
func (s *marketService) GetMarket(ctx context.Context, id string) (*models.Market, error) {
	return s.repo.GetByID(ctx, id)
}

