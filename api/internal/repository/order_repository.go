package repository

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderRepository handles order persistence.
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates an OrderRepository.
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// OrderFilter contains optional filters when listing orders.
type OrderFilter struct {
	UserID   string
	MarketID string
	Status   string
}

// Create persists an order.
func (r *OrderRepository) Create(ctx context.Context, o *models.Order) error {
	return r.db.WithContext(ctx).Create(o).Error
}

// List returns orders matching the provided filter.
// Empty fields in the filter are ignored.
func (r *OrderRepository) List(ctx context.Context, f OrderFilter) ([]models.Order, error) {
	tx := r.db.WithContext(ctx).Model(&models.Order{})
	if f.UserID != "" {
		tx = tx.Where("user_id = ?", f.UserID)
	}
	if f.MarketID != "" {
		tx = tx.Where("market_id = ?", f.MarketID)
	}
	if f.Status != "" {
		tx = tx.Where("status = ?", f.Status)
	}
	var out []models.Order
	if err := tx.Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// GetByID returns an order by ID.
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	var o models.Order
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// Update updates an order.
func (r *OrderRepository) Update(ctx context.Context, o *models.Order) error {
	return r.db.WithContext(ctx).Save(o).Error
}

// Delete soft-deletes an order (cancel).
func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	return r.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("status", models.OrderStatusCancelled).Error
}
