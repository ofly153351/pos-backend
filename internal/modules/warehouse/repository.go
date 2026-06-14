package warehouse

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ProductTotalQtyInWarehouse(ctx context.Context, warehouseID, productID string) (int, error)
	ProductBelongsToStore(ctx context.Context, storeID, productID string) (bool, error)

	// TransferStock transfers stock between warehouses or to a sale_point location.
	TransferStock(ctx context.Context, storeID, sourceWarehouseID, productID string, qty int, destWarehouseID, destinationStoreID, note, createdBy string) error

	// ListWarehouseInventory lists all inventory items for a warehouse (warehouse_inventory table).
	ListWarehouseInventory(ctx context.Context, warehouseID string) ([]WarehouseInventory, error)

	// AllocateInventoryToStock moves quantity from warehouse_inventory to stocks (sale point).
	AllocateInventoryToStock(ctx context.Context, storeID, warehouseID, productID string, qty int, note, createdBy string) error
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
	// Phase W0 safety: never let a delete cascade away locations/stock/history.
	// stocks.location_id and locations.warehouse_id are ON DELETE CASCADE, so a raw
	// delete would silently destroy stock. Reject if anything still references it.
	inUse, err := r.warehouseHasReferences(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrWarehouseInUse
	}
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

// warehouseHasReferences reports whether a warehouse still has dependent data that
// a delete would destroy or orphan. A single location is enough to block: stocks,
// movements and product default-location references all hang off locations.
func (r PostgresRepository) warehouseHasReferences(ctx context.Context, warehouseID string) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Table("locations").Where("warehouse_id = ?", warehouseID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := r.db.WithContext(ctx).Table("warehouse_inventory").Where("warehouse_id = ?", warehouseID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := r.db.WithContext(ctx).Table("warehouse_receipts").Where("warehouse_id = ?", warehouseID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// ProductTotalQtyInWarehouse sums a product's stock across ALL locations of the
// warehouse (sale-point and storage alike) — used to block destructive removal.
func (r PostgresRepository) ProductTotalQtyInWarehouse(ctx context.Context, warehouseID, productID string) (int, error) {
	var total int
	err := r.db.WithContext(ctx).
		Table("stocks").
		Select("COALESCE(SUM(stocks.quantity), 0)").
		Joins("JOIN locations ON locations.id = stocks.location_id").
		Where("locations.warehouse_id = ? AND stocks.product_id = ?", warehouseID, productID).
		Scan(&total).Error
	return total, err
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
		Where("locations.warehouse_id = ? AND locations.is_sale_point = ?", warehouseID, false).
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
	// Self-guarding: only ever delete EMPTY stock rows. Even if a positive-qty row
	// arrives concurrently (in the window between the service's zero-on-hand check
	// and this delete), it survives — closing the silent stock-loss race rather than
	// relying solely on the non-atomic service-layer check.
	result := r.db.WithContext(ctx).
		Exec(`DELETE FROM stocks WHERE product_id = ? AND quantity = 0 AND location_id IN (
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
		Where("locations.warehouse_id = ? AND locations.is_sale_point = ? AND stocks.product_id = ?", warehouseID, false, productID).
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
		Where("stocks.product_id = ? AND locations.warehouse_id = ? AND locations.is_sale_point = ?", productID, warehouseID, false).
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

// cloneOrFindProductType ensures a product_type exists in targetStoreID.
// Matches by name; creates a copy if missing. Returns "" when sourceTypeID is empty.
func (r PostgresRepository) cloneOrFindProductType(ctx context.Context, targetStoreID, sourceTypeID string) (string, error) {
	if sourceTypeID == "" {
		return "", nil
	}
	type row struct {
		Name        string  `gorm:"column:name"`
		Slug        string  `gorm:"column:slug"`
		Description *string `gorm:"column:description"`
	}
	var src row
	if err := r.db.WithContext(ctx).Table("product_types").
		Select("name, slug, description").Where("id = ?", sourceTypeID).Take(&src).Error; err != nil {
		return "", nil // source type gone; skip rather than fail
	}
	var existing struct{ ID string `gorm:"column:id"` }
	err := r.db.WithContext(ctx).Table("product_types").Select("id").
		Where("store_id = ? AND LOWER(name) = LOWER(?)", targetStoreID, src.Name).
		Take(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	// Ensure slug is unique in target store
	slug := src.Slug
	var conflict struct{ ID string `gorm:"column:id"` }
	if r.db.WithContext(ctx).Table("product_types").Select("id").
		Where("store_id = ? AND slug = ?", targetStoreID, slug).Take(&conflict).Error == nil {
		slug = slug + "-" + newID()[:6]
	}
	newTypeID := newID()
	now := time.Now().UTC()
	payload := map[string]any{
		"id": newTypeID, "store_id": targetStoreID,
		"name": src.Name, "slug": slug,
		"is_active": true, "created_at": now, "updated_at": now,
	}
	if src.Description != nil {
		payload["description"] = *src.Description
	}
	if err := r.db.WithContext(ctx).Table("product_types").Create(payload).Error; err != nil {
		return "", err
	}
	return newTypeID, nil
}

// cloneOrFindProductUnit ensures a product_unit exists in targetStoreID.
// Matches by name; creates a copy if missing. Returns an error when sourceUnitID is
// non-empty but cannot be resolved, because product_unit_id is NOT NULL in products.
func (r PostgresRepository) cloneOrFindProductUnit(ctx context.Context, targetStoreID, sourceUnitID string) (string, error) {
	if sourceUnitID == "" {
		return "", nil
	}
	type row struct {
		Name        string  `gorm:"column:name"`
		Description *string `gorm:"column:description"`
	}
	var src row
	if err := r.db.WithContext(ctx).Table("product_units").
		Select("name, description").Where("id = ?", sourceUnitID).Take(&src).Error; err != nil {
		return "", err
	}
	var existingUnit struct{ ID string `gorm:"column:id"` }
	err := r.db.WithContext(ctx).Table("product_units").Select("id").
		Where("store_id = ? AND LOWER(name) = LOWER(?)", targetStoreID, src.Name).
		Take(&existingUnit).Error
	if err == nil {
		return existingUnit.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	newUnitID := newID()
	now := time.Now().UTC()
	payload := map[string]any{
		"id": newUnitID, "store_id": targetStoreID,
		"name": src.Name, "is_active": true,
		"created_at": now, "updated_at": now,
	}
	if src.Description != nil {
		payload["description"] = *src.Description
	}
	if err := r.db.WithContext(ctx).Table("product_units").Create(payload).Error; err != nil {
		return "", err
	}
	return newUnitID, nil
}

// cloneOrFindProductBrand ensures a product_brand exists in targetStoreID.
// Matches by name; creates a copy if missing. Returns "" when sourceBrandID is empty.
func (r PostgresRepository) cloneOrFindProductBrand(ctx context.Context, targetStoreID, sourceBrandID string) (string, error) {
	if sourceBrandID == "" {
		return "", nil
	}
	var srcBrand struct{ Name string `gorm:"column:name"` }
	if err := r.db.WithContext(ctx).Table("product_brands").Select("name").
		Where("id = ?", sourceBrandID).Take(&srcBrand).Error; err != nil {
		return "", nil
	}
	var existingBrand struct{ ID string `gorm:"column:id"` }
	err := r.db.WithContext(ctx).Table("product_brands").Select("id").
		Where("store_id = ? AND LOWER(name) = LOWER(?)", targetStoreID, srcBrand.Name).
		Take(&existingBrand).Error
	if err == nil {
		return existingBrand.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	newBrandID := newID()
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).Table("product_brands").Create(map[string]any{
		"id": newBrandID, "store_id": targetStoreID,
		"name": srcBrand.Name, "is_active": true,
		"created_at": now, "updated_at": now,
	}).Error; err != nil {
		return "", err
	}
	return newBrandID, nil
}

// cloneOrFindProduct ensures a product exists in targetStoreID. It matches by
// barcode first, then SKU. If no match, it clones the product from sourceProductID.
// Returns the product ID in targetStoreID.
func (r PostgresRepository) cloneOrFindProduct(ctx context.Context, targetStoreID, sourceProductID string) (string, error) {
	type productRow struct {
		ID                 string   `gorm:"column:id"`
		Name               string   `gorm:"column:name"`
		SKU                *string  `gorm:"column:sku"`
		Barcode            *string  `gorm:"column:barcode"`
		BasePrice          float64  `gorm:"column:base_price"`
		CostPrice          float64  `gorm:"column:cost_price"`
		SpecialPrice       *float64 `gorm:"column:special_price"`
		SpecialPriceStart  *string  `gorm:"column:special_price_start_at"`
		SpecialPriceEnd    *string  `gorm:"column:special_price_end_at"`
		ProductTypeID      *string  `gorm:"column:product_type_id"`
		ProductUnitID      *string  `gorm:"column:product_unit_id"`
		BrandID            *string  `gorm:"column:brand_id"`
		MinStock           int      `gorm:"column:min_stock"`
		ProductCode        *string  `gorm:"column:product_code"`
		Description        *string  `gorm:"column:description"`
		StorageLocation    *string  `gorm:"column:storage_location"`
	}

	var src productRow
	if err := r.db.WithContext(ctx).
		Table("products").
		Select("id, name, sku, barcode, base_price, cost_price, special_price, special_price_start_at, special_price_end_at, product_type_id, product_unit_id, brand_id, min_stock, product_code, description, storage_location").
		Where("id = ?", sourceProductID).
		Take(&src).Error; err != nil {
		return "", err
	}

	// Match by barcode (preferred), then SKU
	if src.Barcode != nil && *src.Barcode != "" {
		var existingID string
		err := r.db.WithContext(ctx).
			Table("products").
			Select("id").
			Where("store_id = ? AND barcode = ?", targetStoreID, *src.Barcode).
			Take(&existingID).Error
		if err == nil {
			return existingID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}
	if src.SKU != nil && *src.SKU != "" {
		var existingID string
		err := r.db.WithContext(ctx).
			Table("products").
			Select("id").
			Where("store_id = ? AND sku = ?", targetStoreID, *src.SKU).
			Take(&existingID).Error
		if err == nil {
			return existingID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}

	// Clone type, unit, brand into target store first (all idempotent by name/code)
	var destTypeID, destUnitID, destBrandID string
	if src.ProductTypeID != nil {
		id, err := r.cloneOrFindProductType(ctx, targetStoreID, *src.ProductTypeID)
		if err != nil {
			return "", err
		}
		destTypeID = id
	}
	if src.ProductUnitID != nil {
		id, err := r.cloneOrFindProductUnit(ctx, targetStoreID, *src.ProductUnitID)
		if err != nil {
			return "", err
		}
		destUnitID = id
	}
	if src.BrandID != nil {
		id, err := r.cloneOrFindProductBrand(ctx, targetStoreID, *src.BrandID)
		if err != nil {
			return "", err
		}
		destBrandID = id
	}

	// Clone product into target store
	clonedID := newID()
	now := time.Now().UTC()
	payload := map[string]any{
		"id":         clonedID,
		"store_id":   targetStoreID,
		"name":       src.Name,
		"base_price": src.BasePrice,
		"cost_price": src.CostPrice,
		"min_stock":  src.MinStock,
		"is_active":  true,
		"created_at": now,
		"updated_at": now,
	}
	if src.SKU != nil && *src.SKU != "" {
		payload["sku"] = *src.SKU
	}
	if src.Barcode != nil && *src.Barcode != "" {
		payload["barcode"] = *src.Barcode
	}
	if destTypeID != "" {
		payload["product_type_id"] = destTypeID
	}
	// product_unit_id is NOT NULL — must always be set
	if destUnitID == "" {
		return "", fmt.Errorf("could not resolve product unit for product %s in target store", sourceProductID)
	}
	payload["product_unit_id"] = destUnitID
	if destBrandID != "" {
		payload["brand_id"] = destBrandID
	}
	if src.SpecialPrice != nil {
		payload["special_price"] = *src.SpecialPrice
	}
	if src.SpecialPriceStart != nil {
		payload["special_price_start_at"] = *src.SpecialPriceStart
	}
	if src.SpecialPriceEnd != nil {
		payload["special_price_end_at"] = *src.SpecialPriceEnd
	}
	if src.ProductCode != nil {
		payload["product_code"] = *src.ProductCode
	}
	if src.Description != nil {
		payload["description"] = *src.Description
	}
	if src.StorageLocation != nil {
		payload["storage_location"] = *src.StorageLocation
	}

	if err := r.db.WithContext(ctx).Table("products").Create(payload).Error; err != nil {
		return "", err
	}
	return clonedID, nil
}

func (r PostgresRepository) TransferStock(ctx context.Context, storeID, sourceWarehouseID, productID string, qty int, destWarehouseID, destinationStoreID, note, createdBy string) error {
	// 1. Find the first active location in source warehouse
	var srcLoc struct {
		ID string `gorm:"column:id"`
	}
	// Find the location in the source warehouse that actually holds stock for this product.
	// Prefer non-sale-point (warehouse storage) locations, fall back to any location with stock.
	if err := r.db.WithContext(ctx).
		Table("stocks").
		Select("stocks.location_id AS id").
		Joins("JOIN locations ON locations.id = stocks.location_id").
		Where("locations.warehouse_id = ? AND locations.is_active = ? AND stocks.product_id = ? AND stocks.quantity >= ?",
			sourceWarehouseID, true, productID, qty).
		Order("locations.is_sale_point ASC"). // prefer warehouse (non-sale-point) first
		Take(&srcLoc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInsufficientStock
		}
		return err
	}

	// Determine target store for destination stock
	targetStoreID := storeID
	if destinationStoreID != "" {
		targetStoreID = destinationStoreID
	}

	// For cross-store transfer (different target store), use warehouse_inventory system.
	// Do NOT touch stocks table directly — stock must be allocated later via Allocate endpoint.
	var destProductID = productID
	var destLocationID string
	if destinationStoreID != "" && destinationStoreID != storeID {
		// Look up source warehouse info for naming
		var srcWH struct {
			Name string `gorm:"column:name"`
		}
		if err := r.db.WithContext(ctx).
			Table("warehouses").
			Select("name").
			Where("id = ?", sourceWarehouseID).
			Take(&srcWH).Error; err != nil {
			return err
		}

		// Look up source store name
		var srcStoreName string
		if err := r.db.WithContext(ctx).
			Table("stores").
			Select("name").
			Where("id = ?", storeID).
			Take(&srcStoreName).Error; err != nil {
			srcStoreName = storeID
		}

		// Check if warehouse already exists in target store with matching source tracking
		var existingWH struct {
			ID string `gorm:"column:id"`
		}
		err := r.db.WithContext(ctx).
			Table("warehouses").
			Select("id").
			Where("store_id = ? AND source_store_id = ? AND source_warehouse_id = ?", targetStoreID, storeID, sourceWarehouseID).
			Take(&existingWH).Error
		destWarehouseForInventory := ""
		if err == nil {
			destWarehouseForInventory = existingWH.ID
		} else {
			// Create new warehouse in target store with source tracking
			destWarehouseForInventory = newID()
			now := time.Now().UTC()
			whName := srcWH.Name + " (จาก " + srcStoreName + ")"
			if err := r.db.WithContext(ctx).Table("warehouses").Create(map[string]any{
				"id":                 destWarehouseForInventory,
				"store_id":           targetStoreID,
				"name":               whName,
				"is_active":          true,
				"source_store_id":    storeID,
				"source_warehouse_id": sourceWarehouseID,
				"created_at":         now,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}
		}

		// Clone product to target store so Store B owns the product in their catalog.
		destProductID, err = r.cloneOrFindProduct(ctx, targetStoreID, productID)
		if err != nil {
			return err
		}

		// Transaction: deduct from Store A → add directly to Store B's warehouse stocks.
		// No warehouse_inventory staging — product is immediately visible in Store B's warehouse.
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			now := time.Now().UTC()

			// a. Deduct from source location in Store A
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

			// b. Find or create a non-sale-point storage location in destination warehouse
			var destLoc struct{ ID string `gorm:"column:id"` }
			if err := tx.Table("locations").Select("id").
				Where("warehouse_id = ? AND is_sale_point = ? AND is_active = ?", destWarehouseForInventory, false, true).
				Order("created_at ASC").Take(&destLoc).Error; err != nil {
				destLoc.ID = newID()
				if err := tx.Table("locations").Create(map[string]any{
					"id":            destLoc.ID,
					"store_id":      targetStoreID,
					"warehouse_id":  destWarehouseForInventory,
					"name":          "คลังสินค้า",
					"is_sale_point": false,
					"is_active":     true,
					"created_at":    now,
					"updated_at":    now,
				}).Error; err != nil {
					return err
				}
			}

			// c. Upsert into Store B's stocks (visible in warehouse section immediately)
			if err := tx.Exec(`
				INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, NOW(), NOW())
				ON CONFLICT (product_id, location_id)
				DO UPDATE SET quantity = stocks.quantity + ?, updated_at = NOW()
			`, newID(), targetStoreID, destProductID, destLoc.ID, qty, qty).Error; err != nil {
				return err
			}

			// d. OUT movement for Store A
			if err := tx.Table("stock_movements").Create(map[string]any{
				"id":              newID(),
				"store_id":        storeID,
				"product_id":      productID,
				"location_id":     srcLoc.ID,
				"quantity_change": -qty,
				"type":            "TRANSFER",
				"note":            note,
				"created_by":      createdBy,
				"created_at":      now,
				"updated_at":      now,
			}).Error; err != nil {
				return err
			}

			// e. IN movement for Store B (in warehouse storage location)
			if err := tx.Table("stock_movements").Create(map[string]any{
				"id":              newID(),
				"store_id":        targetStoreID,
				"product_id":      destProductID,
				"location_id":     destLoc.ID,
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
	} else if destWarehouseID == "stock" {
		// 2a. Find first is_sale_point location in the target store
		var salePoint struct {
			ID string `gorm:"column:id"`
		}
		if err := r.db.WithContext(ctx).
			Table("locations").
			Select("id").
			Where("store_id = ? AND is_sale_point = ? AND is_active = ?", targetStoreID, true, true).
			Order("created_at ASC").
			Take(&salePoint).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Auto-create a sale point location in the target store
				var whID string
				if err2 := r.db.WithContext(ctx).
					Table("warehouses").
					Where("store_id = ? AND is_active = ?", targetStoreID, true).
					Order("created_at ASC").
					Select("id").
					Take(&whID).Error; err2 != nil {
					// Auto-create a warehouse for the target store
					whID = newID()
					now := time.Now().UTC()
					if err2 := r.db.WithContext(ctx).
						Table("warehouses").
						Create(map[string]any{
							"id":         whID,
							"store_id":   targetStoreID,
							"name":       "คลังหลัก",
							"is_active":  true,
							"created_at": now,
							"updated_at": now,
						}).Error; err2 != nil {
						return err2
					}
				}
				salePoint.ID = newID()
				if err2 := r.db.WithContext(ctx).
					Table("locations").
					Create(map[string]any{
						"id":            salePoint.ID,
						"store_id":      targetStoreID,
						"warehouse_id":  whID,
						"name":          "หน้าร้าน",
						"is_sale_point": true,
						"is_active":     true,
						"created_at":    time.Now().UTC(),
						"updated_at":    time.Now().UTC(),
					}).Error; err2 != nil {
					return err2
				}
			} else {
				return err
			}
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
		`, stockID, targetStoreID, destProductID, destLocationID, qty, qty).Error; err != nil {
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
			"store_id":        targetStoreID,
			"product_id":      destProductID,
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

// ──────────────────────────────────────────────
// Warehouse Inventory methods (warehouse_inventory table)
// ──────────────────────────────────────────────

func (r PostgresRepository) ListWarehouseInventory(ctx context.Context, warehouseID string) ([]WarehouseInventory, error) {
	var items []WarehouseInventory
	err := r.db.WithContext(ctx).
		Table("warehouse_inventory wi").
		Select(`
			wi.id,
			wi.store_id,
			wi.warehouse_id,
			wi.product_id,
			wi.quantity,
			wi.source_store_id,
			wi.source_warehouse_id,
			wi.transferred_at,
			wi.created_at,
			wi.updated_at,
			COALESCE(pv.name, '') AS product_name,
			COALESCE(pv.sku, '') AS product_sku,
			COALESCE(s.name, '') AS source_store_name
		`).
		Joins("LEFT JOIN product_view pv ON pv.id = wi.product_id").
		Joins("LEFT JOIN stores s ON s.id = wi.source_store_id").
		Where("wi.warehouse_id = ?", warehouseID).
		Order("wi.transferred_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []WarehouseInventory{}
	}
	return items, nil
}

func (r PostgresRepository) AllocateInventoryToStock(ctx context.Context, storeID, warehouseID, productID string, qty int, note, createdBy string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock and deduct from warehouse_inventory
		var inv struct {
			ID       string `gorm:"column:id"`
			Quantity int    `gorm:"column:quantity"`
		}
		if err := tx.
			Table("warehouse_inventory").
			Select("id, quantity").
			Where("warehouse_id = ? AND product_id = ?", warehouseID, productID).
			// Lock row for update to prevent race conditions
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Take(&inv).Error; err != nil {
			return ErrInventoryNotFound
		}
		if inv.Quantity < qty {
			return ErrInventoryInsufficientQty
		}

		// Deduct
		result := tx.Exec(`
			UPDATE warehouse_inventory SET quantity = quantity - ?, updated_at = NOW()
			WHERE id = ? AND quantity >= ?
		`, qty, inv.ID, qty)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrInventoryInsufficientQty
		}

		// 2. Find/create sale_point location in the warehouse
		var salePoint struct {
			ID string `gorm:"column:id"`
		}
		err := tx.
			Table("locations").
			Select("id").
			Where("warehouse_id = ? AND is_sale_point = ? AND is_active = ?", warehouseID, true, true).
			Order("created_at ASC").
			Take(&salePoint).Error
		if err != nil {
			// Auto-create a sale_point location named "หน้าร้าน"
			salePoint.ID = newID()
			now := time.Now().UTC()
			if err := tx.Table("locations").Create(map[string]any{
				"id":            salePoint.ID,
				"store_id":      storeID,
				"warehouse_id":  warehouseID,
				"name":          "หน้าร้าน",
				"is_sale_point": true,
				"is_active":     true,
				"created_at":    now,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
		}

		// 3. Upsert into stocks at sale_point location
		stockID := newID()
		if err := tx.Exec(`
			INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())
			ON CONFLICT (product_id, location_id)
			DO UPDATE SET quantity = stocks.quantity + ?, updated_at = NOW()
		`, stockID, storeID, productID, salePoint.ID, qty, qty).Error; err != nil {
			return err
		}

		// 4. Create stock_movement record with type="ALLOCATE" and inventory_id
		movementID := newID()
		now := time.Now().UTC()
		if err := tx.Table("stock_movements").Create(map[string]any{
			"id":              movementID,
			"store_id":        storeID,
			"product_id":      productID,
			"location_id":     salePoint.ID,
			"quantity_change": qty,
			"type":            "ALLOCATE",
			"note":            note,
			"created_by":      createdBy,
			"inventory_id":    inv.ID,
			"created_at":      now,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}
