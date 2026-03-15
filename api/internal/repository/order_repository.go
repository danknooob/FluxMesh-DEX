package repository

import (
	"context"
	"time"

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

// Delete cancels an order via the guarded stored function.
// Returns the cancelled order on success, or a Postgres exception
// containing ORDER_NOT_FOUND / ORDER_NOT_CANCELLABLE on failure.
func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID, userID string) (*models.Order, error) {
	var o models.Order
	result := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_cancel_order($1,$2)", id.String(), userID).
		Scan(&o)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &o, nil
}

// CreateAtomic atomically checks the idempotency key and inserts the order.
// On a duplicate key (including race conditions), it returns the existing
// order and isDuplicate=true instead of an error.
func (r *OrderRepository) CreateAtomic(ctx context.Context, o *models.Order) (isDuplicate bool, err error) {
	type row struct {
		ID             uuid.UUID      `gorm:"column:id"`
		IdempotencyKey *string        `gorm:"column:idempotency_key"`
		UserID         string         `gorm:"column:user_id"`
		MarketID       string         `gorm:"column:market_id"`
		Side           string         `gorm:"column:side"`
		Type           string         `gorm:"column:type"`
		Price          string         `gorm:"column:price"`
		Size           string         `gorm:"column:size"`
		Remaining      string         `gorm:"column:remaining"`
		Status         string         `gorm:"column:status"`
		CancelFee      string         `gorm:"column:cancel_fee"`
		CreatedAt      time.Time      `gorm:"column:created_at"`
		UpdatedAt      time.Time      `gorm:"column:updated_at"`
		DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at"`
		IsDuplicate    bool           `gorm:"column:is_duplicate"`
	}

	// Function returns one row: either existing (idempotency) or newly inserted. SELECT * fetches all returned columns.
	var rows []row
	if err := r.db.WithContext(ctx).
		Raw("SELECT * FROM fn_create_order_atomic($1,$2,$3,$4,$5,$6,$7,$8)",
			o.IdempotencyKey, o.UserID, o.MarketID,
			string(o.Side), string(o.Type),
			o.Price, o.Size, o.Remaining,
		).Scan(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, gorm.ErrRecordNotFound
	}

	res := rows[0]
	o.ID = res.ID
	if res.IdempotencyKey != nil {
		o.IdempotencyKey = *res.IdempotencyKey
	}
	o.UserID = res.UserID
	o.MarketID = res.MarketID
	o.Side = models.OrderSide(res.Side)
	o.Type = models.OrderType(res.Type)
	o.Price = res.Price
	o.Size = res.Size
	o.Remaining = res.Remaining
	o.Status = models.OrderStatus(res.Status)
	o.CancelFee = res.CancelFee
	o.CreatedAt = res.CreatedAt
	o.UpdatedAt = res.UpdatedAt
	o.DeletedAt = res.DeletedAt

	return res.IsDuplicate, nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
