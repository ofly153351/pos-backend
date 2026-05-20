package warehouse

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, item Warehouse) (Warehouse, error)
	ListByStore(ctx context.Context, storeID string) ([]Warehouse, error)
	GetByID(ctx context.Context, storeID, id string) (Warehouse, error)
	Update(ctx context.Context, item Warehouse) (Warehouse, error)
	Delete(ctx context.Context, storeID, id string) error
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)

	// Warehouse-Product association (via stocks + locations)
	AddProduct(ctx context.Context, storeID, warehouseID, productID string, quantity int) (WarehouseProduct, error)
	ListProducts(ctx context.Context, warehouseID string) ([]WarehouseProduct, error)
	UpdateProduct(ctx context.Context, storeID, warehouseID, productID string, quantity int) error
	RemoveProduct(ctx context.Context, storeID, warehouseID, productID string) error
	ProductExistsInWarehouse(ctx context.Context, warehouseID, productID string) (bool, error)
	ProductBelongsToStore(ctx context.Context, storeID, productID string) (bool, error)

	// TransferStock transfers stock between warehouses or to a sale_point location.
	TransferStock(ctx context.Context, storeID, sourceWarehouseID, productID string, qty int, destWarehouseID, note, createdBy string) error
}

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) PostgresRepository { return PostgresRepository{db: db} }

func (r PostgresRepository) Create(ctx context.Context, item Warehouse) (Warehouse, error) {
	payload := map[string]any{
		"id":           item.ID,
		"store_id":     item.StoreID,
		"name":         item.Name,
		"code":         nilEmpty(item.Code),
		"address":      nilEmpty(item.Address),
		"phone":        nilEmpty(item.Phone),
		"contact_name": nilEmpty(item.ContactName),
		"is_active":    item.IsActive,
		"created_at":   item.CreatedAt,
		"updated_at":   item.CreatedAt,
	}
	err := r.db.WithContext(ctx).Table("warehouses").Create(payload).Error
	if err != nil {
		return Warehouse{}, err
	}
	item.UpdatedAt = item.CreatedAt
	return item, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Warehouse, error) {
	var items []Warehouse
	err := r.db.WithContext(ctx).
		Model(&Warehouse{}).
		Where("store_id = ?", storeID).
		Order("name ASC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, id string) (Warehouse, error) {
	var item Warehouse
	err := r.db.WithContext(ctx).
		Model(&Warehouse{}).
		Where("store_id = ? AND id = ?", storeID, id).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Warehouse{}, ErrWarehouseNotFound
		}
		return Warehouse{}, err
	}
	return item, nil
}

func (r PostgresRepository) Update(ctx context.Context, item Warehouse) (Warehouse, error) {
	updates := map[string]any{
		"name":         item.Name,
		"code":         nilEmpty(item.Code),
		"address":      nilEmpty(item.Address),
		"phone":        nilEmpty(item.Phone),
		"contact_name": nilEmpty(item.ContactName),
		"is_active":    item.IsActive,
		"updated_at":   item.UpdatedAt,
	}
	result := r.db.WithContext(ctx).
		Model(&Warehouse{}).
		Where("store_id = ? AND id = ?", item.StoreID, item.ID).
		Updates(updates)
	if result.Error != nil {
		return Warehouse{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Warehouse{}, ErrWarehouseNotFound
	}
	return item, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, id string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, id).
		Delete(&Warehouse{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWarehouseNotFound
	}
	return nil
}

func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func nilEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ──────────────────────────────────────────────
// Warehouse-Product repository methods (stocks + locations)
// ──────────────────────────────────────────────

func (r PostgresRepository) AddProduct(ctx context.Context, storeID, warehouseID, productID string, quantity int) (WarehouseProduct, error) {
	// Ensure at least one active location exists in the warehouse.
	// If none, auto-create a default location called "คลังหลัก".
	var locationID string
	var loc struct {
		ID string `gorm:"column:id"`
	}
	err := r.db.WithContext(ctx).
		Table("locations").
		Select("id").
		Where("warehouse_id = ? AND is_active = ?", warehouseID, true).
		Order("created_at ASC").
		Take(&loc).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return WarehouseProduct{}, err
		}
		// No location exists — create default
		locationID = newID()
		now := time.Now().UTC()
			err = r.db.WithContext(ctx).Table("locations").Create(map[string]any{
				"id":            locationID,
				"store_id":      storeID,
				"warehouse_id":  warehouseID,
				"name":          "คลังหลัก",
				"is_sale_point": false,
				"is_active":     true,
				"created_at":    now,
				"updated_at":    now,
			}).Error
		if err != nil {
			return WarehouseProduct{}, err
		}
	} else {
		locationID = loc.ID
	}

	// Upsert stock — add quantity to existing or insert new row
	stockID := newID()
	err = r.db.WithContext(ctx).Exec(`
		INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (product_id, location_id)
		DO UPDATE SET quantity = stocks.quantity + ?, updated_at = NOW()
	`, stockID, storeID, productID, locationID, quantity, quantity).Error
	if err != nil {
		return WarehouseProduct{}, err
	}

	return r.getWarehouseProduct(ctx, productID, warehouseID)
}

func (r PostgresRepository) ListProducts(ctx context.Context, warehouseID string) ([]WarehouseProduct, error) {
	var items []WarehouseProduct
	err := r.db.WithContext(ctx).
		Table("stocks").
		Select(`
			stocks.product_id,
			COALESCE(pv.name, '') AS product_name,
			COALESCE(pv.sku, '') AS product_sku,
			COALESCE(pv.barcode, '') AS product_barcode,
			COALESCE(pv.base_price, 0) AS product_price,
			COALESCE(pv.cost_price, 0) AS cost_price,
			CASE
				WHEN pv.special_price IS NOT NULL
					AND (pv.special_price_start_at IS NULL OR pv.special_price_start_at <= NOW())
					AND (pv.special_price_end_at IS NULL OR pv.special_price_end_at >= NOW())
				THEN pv.special_price
				ELSE pv.base_price
			END AS effective_price,
			COALESCE(pv.image_url, '') AS image_url,
			COALESCE(pv.product_type_name, '') AS product_type_name,
			COALESCE(pv.product_unit_name, '') AS product_unit_name,
			COALESCE(pv.min_stock, 0) AS product_min_stock,
			pv.max_stock AS product_max_stock,
			SUM(stocks.quantity) AS quantity
		`).
		Joins("JOIN locations ON locations.id = stocks.location_id").
		Joins("LEFT JOIN product_view pv ON pv.id = stocks.product_id").
		Where("locations.warehouse_id = ?", warehouseID).
		Group("stocks.product_id, pv.name, pv.sku, pv.barcode, pv.base_price, pv.cost_price, pv.special_price, pv.special_price_start_at, pv.special_price_end_at, pv.image_url, pv.product_type_name, pv.product_unit_name, pv.min_stock, pv.max_stock").
		Order("COALESCE(pv.name, '') ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []WarehouseProduct{}
	}
	return items, nil
}

func (r PostgresRepository) UpdateProduct(ctx context.Context, storeID, warehouseID, productID string, quantity int) error {
	// Find the first active location in the warehouse
	var loc struct {
		ID string `gorm:"column:id"`
	}
	err := r.db.WithContext(ctx).
		Table("locations").
		Select("id").
		Where("warehouse_id = ? AND is_active = ?", warehouseID, true).
		Order("created_at ASC").
		Take(&loc).Error
	if err != nil {
		return ErrProductNotInWarehouse
	}

	// Remove stock entries for this product at all OTHER locations in the warehouse
	err = r.db.WithContext(ctx).
		Exec(`DELETE FROM stocks WHERE product_id = ? AND location_id IN (
			SELECT id FROM locations WHERE warehouse_id = ? AND id != ?
		)`, productID, warehouseID, loc.ID).Error
	if err != nil {
		return err
	}

	// Upsert the total quantity at the first location
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (product_id, location_id)
		DO UPDATE SET quantity = ?, updated_at = NOW()
	`, newID(), storeID, productID, loc.ID, quantity, quantity).Error
}

func (r PostgresRepository) RemoveProduct(ctx context.Context, storeID, warehouseID, productID string) error {
	result := r.db.WithContext(ctx).
		Exec(`DELETE FROM stocks WHERE product_id = ? AND location_id IN (
			SELECT id FROM locations WHERE warehouse_id = ?
		)`, productID, warehouseID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductNotInWarehouse
	}
	return nil
}

func (r PostgresRepository) ProductExistsInWarehouse(ctx context.Context, warehouseID, productID string) (bool, error) {
	var total int
	err := r.db.WithContext(ctx).
		Table("stocks").
		Select("COALESCE(SUM(stocks.quantity), 0)").
		Joins("JOIN locations ON locations.id = stocks.location_id").
		Where("locations.warehouse_id = ? AND stocks.product_id = ?", warehouseID, productID).
		Scan(&total).Error
	if err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r PostgresRepository) ProductBelongsToStore(ctx context.Context, storeID, productID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("products").
		Where("id = ? AND store_id = ?", productID, storeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, ErrProductNotFound
	}
	return true, nil
}

// getWarehouseProduct fetches a single product's aggregated stock info from a warehouse.
func (r PostgresRepository) getWarehouseProduct(ctx context.Context, productID, warehouseID string) (WarehouseProduct, error) {
	var wp WarehouseProduct
	err := r.db.WithContext(ctx).
		Table("stocks").
		Select(`
			stocks.product_id,
			COALESCE(pv.name, '') AS product_name,
			COALESCE(pv.sku, '') AS product_sku,
			COALESCE(pv.barcode, '') AS product_barcode,
			COALESCE(pv.base_price, 0) AS product_price,
			COALESCE(pv.cost_price, 0) AS cost_price,
			CASE
				WHEN pv.special_price IS NOT NULL
					AND (pv.special_price_start_at IS NULL OR pv.special_price_start_at <= NOW())
					AND (pv.special_price_end_at IS NULL OR pv.special_price_end_at >= NOW())
				THEN pv.special_price
				ELSE pv.base_price
			END AS effective_price,
			COALESCE(pv.image_url, '') AS image_url,
			COALESCE(pv.product_type_name, '') AS product_type_name,
			COALESCE(pv.product_unit_name, '') AS product_unit_name,
			COALESCE(pv.min_stock, 0) AS product_min_stock,
			pv.max_stock AS product_max_stock,
			SUM(stocks.quantity) AS quantity
		`).
		Joins("JOIN locations ON locations.id = stocks.location_id").
		Joins("LEFT JOIN product_view pv ON pv.id = stocks.product_id").
		Where("stocks.product_id = ? AND locations.warehouse_id = ?", productID, warehouseID).
		Group("stocks.product_id, pv.name, pv.sku, pv.barcode, pv.base_price, pv.cost_price, pv.special_price, pv.special_price_start_at, pv.special_price_end_at, pv.image_url, pv.product_type_name, pv.product_unit_name, pv.min_stock, pv.max_stock").
		Take(&wp).Error
	if err != nil {
		return WarehouseProduct{}, err
	}
	return wp, nil
}

// ──────────────────────────────────────────────
// Transfer stock methods
// ──────────────────────────────────────────────

func (r PostgresRepository) TransferStock(ctx context.Context, storeID, sourceWarehouseID, productID string, qty int, destWarehouseID, note, createdBy string) error {
	// 1. Find the first active location in source warehouse
	var srcLoc struct {
		ID string `gorm:"column:id"`
	}
	if err := r.db.WithContext(ctx).
		Table("locations").
		Select("id").
		Where("warehouse_id = ? AND is_active = ?", sourceWarehouseID, true).
		Order("created_at ASC").
		Take(&srcLoc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProductNotInWarehouse
		}
		return err
	}

	var destLocationID string
	if destWarehouseID == "stock" {
		// 2a. Find first is_sale_point location in the store
		var salePoint struct {
			ID string `gorm:"column:id"`
		}
		if err := r.db.WithContext(ctx).
			Table("locations").
			Select("id").
			Where("store_id = ? AND is_sale_point = ? AND is_active = ?", storeID, true, true).
			Order("created_at ASC").
			Take(&salePoint).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("no sale point location found in this store")
			}
			return err
		}
		destLocationID = salePoint.ID
	} else {
		// 2b. Find first active location in destination warehouse
		var destLoc struct {
			ID string `gorm:"column:id"`
		}
		if err := r.db.WithContext(ctx).
			Table("locations").
			Select("id").
			Where("warehouse_id = ? AND is_active = ?", destWarehouseID, true).
			Order("created_at ASC").
			Take(&destLoc).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProductNotInWarehouse
			}
			return err
		}
		destLocationID = destLoc.ID
	}

	// 3. Execute transfer in a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// a. Deduct from source location
		result := tx.Exec(`
			UPDATE stocks SET quantity = quantity - ?, updated_at = NOW()
			WHERE product_id = ? AND location_id = ? AND quantity >= ?
		`, qty, productID, srcLoc.ID, qty)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrInsufficientStock
		}

		// b. Upsert to destination location
		stockID := newID()
		if err := tx.Exec(`
			INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())
			ON CONFLICT (product_id, location_id)
			DO UPDATE SET quantity = stocks.quantity + ?, updated_at = NOW()
		`, stockID, storeID, productID, destLocationID, qty, qty).Error; err != nil {
			return err
		}

		now := time.Now().UTC()

		// c. Create OUT movement record
		outID := newID()
		if err := tx.Table("stock_movements").Create(map[string]any{
			"id":                       outID,
			"store_id":                 storeID,
			"product_id":               productID,
			"location_id":              srcLoc.ID,
			"destination_location_id":  destLocationID,
			"quantity_change":          -qty,
			"type":                     "TRANSFER",
			"note":                     note,
			"created_by":               createdBy,
			"created_at":               now,
			"updated_at":               now,
		}).Error; err != nil {
			return err
		}

		// d. Create IN movement record
		inID := newID()
		if err := tx.Table("stock_movements").Create(map[string]any{
			"id":              inID,
			"store_id":        storeID,
			"product_id":      productID,
			"location_id":     destLocationID,
			"reference_id":    srcLoc.ID,
			"quantity_change": qty,
			"type":            "TRANSFER",
			"note":            note,
			"created_by":      createdBy,
			"created_at":      now,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}
