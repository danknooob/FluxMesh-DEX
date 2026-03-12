package repository

import (
	"context"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderRepository handles order persistence via stored functions.
type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// OrderFilter contains optional filters when listing orders.
type OrderFilter struct {
	UserID   string
	MarketID string
	Status   string
}

func (r *OrderRepository) FindByIdempotencyKey(ctx context.Context, key string) (*models.Order, error) {
	if key == "" {
		return nil, nil
	}
	var o models.Order
	result := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_find_order_by_idempotency_key($1)", key).
		Scan(&o)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &o, nil
}

func (r *OrderRepository) Create(ctx context.Context, o *models.Order) error {
	return r.db.WithContext(ctx).
		Raw(
			"SELECT * FROM fn_create_order($1,$2,$3,$4,$5,$6,$7,$8)",
			o.IdempotencyKey, o.UserID, o.MarketID,
			string(o.Side), string(o.Type),
			o.Price, o.Size, o.Remaining,
		).Scan(o).Error
}

func (r *OrderRepository) List(ctx context.Context, f OrderFilter) ([]models.Order, error) {
	userID := nilIfEmpty(f.UserID)
	marketID := nilIfEmpty(f.MarketID)
	status := nilIfEmpty(f.Status)

	var out []models.Order
	err := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_list_orders($1,$2,$3)", userID, marketID, status).
		Scan(&out).Error
	return out, err
}

// PriceLevel is one aggregated row in the order book depth.
type PriceLevel struct {
	Price     string `json:"price"`
	TotalSize string `json:"total_size"`
	Count     int    `json:"count"`
}

func (r *OrderRepository) Depth(ctx context.Context, marketID string, side string, limit int) ([]PriceLevel, error) {
	var levels []PriceLevel
	err := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_order_depth($1,$2,$3)", marketID, side, limit).
		Scan(&levels).Error
	return levels, err
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	var o models.Order
	result := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_get_order_by_id($1)", id.String()).
		Scan(&o)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &o, nil
}

func (r *OrderRepository) Update(ctx context.Context, o *models.Order) error {
	return r.db.WithContext(ctx).
		Exec(
			"SELECT fn_update_order($1,$2,$3,$4,$5,$6,$7,$8,$9)",
			o.ID.String(), o.UserID, o.MarketID,
			string(o.Side), string(o.Type),
			o.Price, o.Size, o.Remaining,
			string(o.Status),
		).Error
}

func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	return r.db.WithContext(ctx).
		Exec("SELECT fn_cancel_order($1,$2)", id.String(), userID).Error
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
