package stock_movement

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, m StockMovement) (StockMovement, error)
	ListByStore(ctx context.Context, storeID string, q ListMovementsQuery) (MovementResponse, error)
	GetProduct(ctx context.Context, storeID, productID string) (ProductRef, error)
	UpdateProductQuantity(ctx context.Context, productID string, addQty int) error
}

type ProductRef struct {
	ID       string
	StoreID  string
	Name     string
	Quantity int
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
			stock_movements.quantity_change,
			stock_movements.type,
			stock_movements.reference_id,
			stock_movements.note,
			stock_movements.created_by,
			stock_movements.created_at,
			stock_movements.updated_at,
			COALESCE(products.name, '') AS product_name,
			COALESCE(products.sku, '') AS product_sku,
			COALESCE(users.full_name, '') AS created_by_name
		`).
		Joins("LEFT JOIN products ON products.id = stock_movements.product_id").
		Joins("LEFT JOIN users ON users.id = stock_movements.created_by").
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

func (r PostgresRepository) GetProduct(ctx context.Context, storeID, productID string) (ProductRef, error) {
	var p ProductRef
	err := r.db.WithContext(ctx).
		Table("products").
		Select("id, store_id, name, quantity").
		Where("id = ? AND store_id = ?", productID, storeID).
		Take(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductRef{}, ErrProductNotFound
		}
		return ProductRef{}, err
	}
	return p, nil
}

func (r PostgresRepository) UpdateProductQuantity(ctx context.Context, productID string, addQty int) error {
	return r.db.WithContext(ctx).
		Table("products").
		Where("id = ?", productID).
		Update("quantity", gorm.Expr("quantity + ?", addQty)).Error
}
