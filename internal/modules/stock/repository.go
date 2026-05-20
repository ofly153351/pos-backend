package stock

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	GetByProduct(ctx context.Context, storeID, productID string) (StockSummary, error)
	GetByLocation(ctx context.Context, storeID, locationID string) ([]Stock, error)
	Upsert(ctx context.Context, storeID, productID, locationID string, delta int) error
	SetQuantity(ctx context.Context, storeID, productID, locationID string, qty int) error
	GetCurrentQuantity(ctx context.Context, storeID, productID, locationID string) (int, error)
	ListLowStock(ctx context.Context, storeID string, threshold int) ([]LowStockItem, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) GetByProduct(ctx context.Context, storeID, productID string) (StockSummary, error) {
	type locationRow struct {
		LocationID    string `gorm:"column:location_id"`
		LocationName  string `gorm:"column:location_name"`
		WarehouseName string `gorm:"column:warehouse_name"`
		Quantity      int    `gorm:"column:quantity"`
	}

	var rows []locationRow
	err := r.db.WithContext(ctx).
		Table("stocks").
		Select(`
			stocks.location_id,
			COALESCE(locations.name, '') AS location_name,
			COALESCE(warehouses.name, '') AS warehouse_name,
			stocks.quantity
		`).
		Joins("LEFT JOIN locations ON locations.id = stocks.location_id").
		Joins("LEFT JOIN warehouses ON warehouses.id = locations.warehouse_id").
		Where("stocks.store_id = ? AND stocks.product_id = ?", storeID, productID).
		Order("locations.name ASC").
		Find(&rows).Error
	if err != nil {
		return StockSummary{}, err
	}

	totalQty := 0
	locations := make([]StockByLocation, 0, len(rows))
	for _, row := range rows {
		totalQty += row.Quantity
		locations = append(locations, StockByLocation{
			LocationID:    row.LocationID,
			LocationName:  row.LocationName,
			WarehouseName: row.WarehouseName,
			Quantity:      row.Quantity,
		})
	}

	return StockSummary{
		ProductID:     productID,
		TotalQuantity: totalQty,
		Locations:     locations,
	}, nil
}

func (r PostgresRepository) GetByLocation(ctx context.Context, storeID, locationID string) ([]Stock, error) {
	var items []Stock
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND location_id = ?", storeID, locationID).
		Order("created_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []Stock{}
	}
	return items, nil
}

func (r PostgresRepository) Upsert(ctx context.Context, storeID, productID, locationID string, delta int) error {
	id := newID()
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

func (r PostgresRepository) SetQuantity(ctx context.Context, storeID, productID, locationID string, qty int) error {
	if qty < 0 {
		return ErrNegativeStock
	}
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
		// Insert new row
		return r.db.WithContext(ctx).
			Table("stocks").
			Create(map[string]any{
				"id":         newID(),
				"store_id":   storeID,
				"product_id": productID,
				"location_id": locationID,
				"quantity":   qty,
				"created_at": gorm.Expr("NOW()"),
				"updated_at": gorm.Expr("NOW()"),
			}).Error
	}
	return nil
}

func (r PostgresRepository) GetCurrentQuantity(ctx context.Context, storeID, productID, locationID string) (int, error) {
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

func (r PostgresRepository) ListLowStock(ctx context.Context, storeID string, threshold int) ([]LowStockItem, error) {
	var items []LowStockItem
	err := r.db.WithContext(ctx).
		Table("stocks").
		Select(`
			stocks.product_id,
			COALESCE(products.name, '') AS product_name,
			COALESCE(products.sku, '') AS sku,
			SUM(stocks.quantity) AS quantity,
			products.min_stock
		`).
		Joins("JOIN products ON products.id = stocks.product_id AND products.store_id = stocks.store_id").
		Where("stocks.store_id = ?", storeID).
		Group("stocks.product_id, products.name, products.sku, products.min_stock").
		Having("SUM(stocks.quantity) <= ?", threshold).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []LowStockItem{}
	}
	return items, nil
}
