package stock_movement

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, m StockMovement) (StockMovement, error)
	ListByStore(ctx context.Context, storeID string, q ListMovementsQuery) (MovementResponse, error)
	UpsertStock(ctx context.Context, storeID, productID, locationID string, delta int) error
	SetStockQuantity(ctx context.Context, storeID, productID, locationID string, qty int) error
	GetCurrentStockQty(ctx context.Context, storeID, productID, locationID string) (int, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, m StockMovement) (StockMovement, error) {
	payload := map[string]any{
		"id":              m.ID,
		"store_id":        m.StoreID,
		"product_id":      m.ProductID,
		"quantity_change": m.QuantityChange,
		"type":            m.Type,
		"note":            m.Note,
		"created_by":      m.CreatedBy,
		"created_at":      m.CreatedAt,
		"updated_at":      m.CreatedAt,
	}
	if m.ReferenceID != nil {
		payload["reference_id"] = *m.ReferenceID
	}
	if m.LocationID != nil {
		payload["location_id"] = *m.LocationID
	}
	if m.DestinationLocationID != nil {
		payload["destination_location_id"] = *m.DestinationLocationID
	}
	if err := r.db.WithContext(ctx).Table("stock_movements").Create(payload).Error; err != nil {
		return StockMovement{}, err
	}
	m.UpdatedAt = m.CreatedAt
	return m, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string, q ListMovementsQuery) (MovementResponse, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	cq := r.db.WithContext(ctx).Table("stock_movements").
		Joins("LEFT JOIN products ON products.id = stock_movements.product_id").
		Joins("LEFT JOIN users ON users.id = stock_movements.created_by").
		Where("stock_movements.store_id = ?", storeID)
	if q.ProductID != "" {
		cq = cq.Where("stock_movements.product_id = ?", q.ProductID)
	}
	if err := cq.Count(&total).Error; err != nil {
		return MovementResponse{}, err
	}

	lq := r.db.WithContext(ctx).Table("stock_movements").
		Select(`
			stock_movements.id,
			stock_movements.store_id,
			stock_movements.product_id,
			stock_movements.location_id,
			stock_movements.destination_location_id,
			stock_movements.quantity_change,
			stock_movements.type,
			stock_movements.reference_id,
			stock_movements.note,
			stock_movements.created_by,
			stock_movements.created_at,
			stock_movements.updated_at,
			COALESCE(products.name, '') AS product_name,
			COALESCE(products.sku, '') AS product_sku,
			COALESCE(users.full_name, '') AS created_by_name,
			COALESCE(locations.name, '') AS location_name
		`).
		Joins("LEFT JOIN products ON products.id = stock_movements.product_id").
		Joins("LEFT JOIN users ON users.id = stock_movements.created_by").
		Joins("LEFT JOIN locations ON locations.id = stock_movements.location_id").
		Where("stock_movements.store_id = ?", storeID)
	if q.ProductID != "" {
		lq = lq.Where("stock_movements.product_id = ?", q.ProductID)
	}

	var items []StockMovement
	if err := lq.Order("stock_movements.created_at DESC").Offset(offset).Limit(limit).Scan(&items).Error; err != nil {
		return MovementResponse{}, err
	}
	if items == nil {
		items = []StockMovement{}
	}

	lastPage := int(total) / limit
	if int(total)%limit > 0 {
		lastPage++
	}
	if lastPage < 1 {
		lastPage = 1
	}

	return MovementResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		Limit:    limit,
		LastPage: lastPage,
	}, nil
}

func (r PostgresRepository) UpsertStock(ctx context.Context, storeID, productID, locationID string, delta int) error {
	id := newStockID()
	result := r.db.WithContext(ctx).
		Exec(`
			INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
			VALUES (?, ?, ?, ?, GREATEST(0, ?), NOW(), NOW())
			ON CONFLICT (product_id, location_id)
			DO UPDATE SET quantity = stocks.quantity + ?, updated_at = NOW()
			WHERE (stocks.quantity + ?) >= 0
		`, id, storeID, productID, locationID, delta, delta, delta)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInsufficientStock
	}
	return nil
}

func (r PostgresRepository) SetStockQuantity(ctx context.Context, storeID, productID, locationID string, qty int) error {
	result := r.db.WithContext(ctx).
		Table("stocks").
		Where("store_id = ? AND product_id = ? AND location_id = ?", storeID, productID, locationID).
		Updates(map[string]any{
			"quantity":   qty,
			"updated_at": gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// No row exists, insert new one
		return r.db.WithContext(ctx).
			Table("stocks").
			Create(map[string]any{
				"id":          newStockID(),
				"store_id":    storeID,
				"product_id":  productID,
				"location_id": locationID,
				"quantity":    qty,
				"created_at":  gorm.Expr("NOW()"),
				"updated_at":  gorm.Expr("NOW()"),
			}).Error
	}
	return nil
}

func (r PostgresRepository) GetCurrentStockQty(ctx context.Context, storeID, productID, locationID string) (int, error) {
	var qty int
	err := r.db.WithContext(ctx).
		Table("stocks").
		Select("COALESCE(quantity, 0)").
		Where("store_id = ? AND product_id = ? AND location_id = ?", storeID, productID, locationID).
		Take(&qty).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return qty, nil
}
