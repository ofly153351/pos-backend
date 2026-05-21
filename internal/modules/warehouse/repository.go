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
	TransferStock(ctx context.Context, storeID, sourceWarehouseID, productID string, qty int, destWarehouseID, destinationStoreID, note, createdBy string) error

	// CloneWarehouseToStore clones a warehouse (with its locations) and all products
	// that have stock in that warehouse to the target store.
	CloneWarehouseToStore(ctx context.Context, sourceWarehouseID, targetStoreID string) (newWarehouseID string, locationIDMap map[string]string, productIDMap map[string]string, err error)
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

func (r PostgresRepository) TransferStock(ctx context.Context, storeID, sourceWarehouseID, productID string, qty int, destWarehouseID, destinationStoreID, note, createdBy string) error {
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

	// Determine target store for destination stock
	targetStoreID := storeID
	if destWarehouseID == "stock" && destinationStoreID != "" {
		targetStoreID = destinationStoreID
	}

	// For cross-store transfer (different target store), clone the source warehouse
	// (including locations and products) to the target store, then switch to warehouse mode.
	// This ensures all locations and products exist in the destination store.
	var destProductID = productID
	if destWarehouseID == "stock" && destinationStoreID != "" && destinationStoreID != storeID {
		clonedWHID, _, prodMap, cloneErr := r.CloneWarehouseToStore(ctx, sourceWarehouseID, targetStoreID)
		if cloneErr != nil {
			return cloneErr
		}
		// Use the cloned warehouse as the destination (warehouse mode)
		destWarehouseID = clonedWHID
		if mappedID, ok := prodMap[productID]; ok {
			destProductID = mappedID
		}
	}

	var destLocationID string
	if destWarehouseID == "stock" {
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

// CloneWarehouseToStore clones a warehouse (with its locations) and all products
// that have stock in that warehouse to the target store.
func (r PostgresRepository) CloneWarehouseToStore(ctx context.Context, sourceWarehouseID, targetStoreID string) (newWarehouseID string, locationIDMap map[string]string, productIDMap map[string]string, err error) {
	locationIDMap = make(map[string]string)
	productIDMap = make(map[string]string)

	// 1. Query source warehouse data
	var srcWarehouse struct {
		Name        string `gorm:"column:name"`
		Code        string `gorm:"column:code"`
		Address     string `gorm:"column:address"`
		Phone       string `gorm:"column:phone"`
		ContactName string `gorm:"column:contact_name"`
		IsActive    bool   `gorm:"column:is_active"`
	}
	if err := r.db.WithContext(ctx).
		Table("warehouses").
		Select("name, code, address, phone, contact_name, is_active").
		Where("id = ?", sourceWarehouseID).
		Take(&srcWarehouse).Error; err != nil {
		return "", nil, nil, err
	}

	// 2. INSERT new warehouse at target store
	now := time.Now().UTC()
	newWarehouseID = newID()
	if err := r.db.WithContext(ctx).Table("warehouses").Create(map[string]any{
		"id":           newWarehouseID,
		"store_id":     targetStoreID,
		"name":         srcWarehouse.Name,
		"code":         nilEmpty(srcWarehouse.Code),
		"address":      nilEmpty(srcWarehouse.Address),
		"phone":        nilEmpty(srcWarehouse.Phone),
		"contact_name": nilEmpty(srcWarehouse.ContactName),
		"is_active":    true,
		"created_at":   now,
		"updated_at":   now,
	}).Error; err != nil {
		return "", nil, nil, err
	}

	// 3. Query source locations
	type srcLocation struct {
		ID          string `gorm:"column:id"`
		Name        string `gorm:"column:name"`
		IsSalePoint bool   `gorm:"column:is_sale_point"`
		IsActive    bool   `gorm:"column:is_active"`
	}
	var srcLocations []srcLocation
	if err := r.db.WithContext(ctx).
		Table("locations").
		Select("id, name, is_sale_point, is_active").
		Where("warehouse_id = ?", sourceWarehouseID).
		Find(&srcLocations).Error; err != nil {
		return "", nil, nil, err
	}

	// 4. INSERT cloned locations at target store
	for _, loc := range srcLocations {
		newLocID := newID()
		if err := r.db.WithContext(ctx).Table("locations").Create(map[string]any{
			"id":            newLocID,
			"store_id":      targetStoreID,
			"warehouse_id":  newWarehouseID,
			"name":          loc.Name,
			"is_sale_point": loc.IsSalePoint,
			"is_active":     loc.IsActive,
			"created_at":    now,
			"updated_at":    now,
		}).Error; err != nil {
			return "", nil, nil, err
		}
		locationIDMap[loc.ID] = newLocID
	}

	// 5. Query distinct product IDs that have stock in the source warehouse
	var srcProductIDs []string
	if err := r.db.WithContext(ctx).
		Table("stocks").
		Select("DISTINCT s.product_id").
		Joins("JOIN locations l ON l.id = s.location_id").
		Where("l.warehouse_id = ?", sourceWarehouseID).
		Pluck("s.product_id", &srcProductIDs).Error; err != nil {
		return "", nil, nil, err
	}
	if len(srcProductIDs) == 0 {
		return newWarehouseID, locationIDMap, productIDMap, nil
	}

	// 6. Clone each product to the target store
	type srcProductData struct {
		Name                string     `gorm:"column:name"`
		SKU                 string     `gorm:"column:sku"`
		Barcode             string     `gorm:"column:barcode"`
		ImageURL            string     `gorm:"column:image_url"`
		BasePrice           float64    `gorm:"column:base_price"`
		CostPrice           float64    `gorm:"column:cost_price"`
		SpecialPrice        *float64   `gorm:"column:special_price"`
		SpecialPriceStartAt *time.Time `gorm:"column:special_price_start_at"`
		SpecialPriceEndAt   *time.Time `gorm:"column:special_price_end_at"`
		IsActive            bool       `gorm:"column:is_active"`
		MinStock            int        `gorm:"column:min_stock"`
		MaxStock            *int       `gorm:"column:max_stock"`
		ProductCode         string     `gorm:"column:product_code"`
		Description         string     `gorm:"column:description"`
		StorageLocation     string     `gorm:"column:storage_location"`
		ProductUnitID       string     `gorm:"column:product_unit_id"`
	}

	for _, srcPID := range srcProductIDs {
		// a. Query source product data
		var pData srcProductData
		if err := r.db.WithContext(ctx).
			Table("products").
			Select("name, sku, barcode, image_url, base_price, cost_price, special_price, special_price_start_at, special_price_end_at, is_active, min_stock, max_stock, product_code, description, storage_location, product_unit_id").
			Where("id = ?", srcPID).
			Take(&pData).Error; err != nil {
			return "", nil, nil, err
		}

		// b. Check if SKU already exists in target store
		if pData.SKU != "" {
			var existingDestID string
			if err := r.db.WithContext(ctx).
				Table("products").
				Select("id").
				Where("store_id = ? AND sku = ?", targetStoreID, pData.SKU).
				Take(&existingDestID).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return "", nil, nil, err
				}
			}
			if existingDestID != "" {
				productIDMap[srcPID] = existingDestID
				continue
			}
		}

		// c. Resolve product_unit_id for target store
		destUnitID := pData.ProductUnitID
		if pData.ProductUnitID != "" {
			// Find unit name from source
			var unitName string
			if err := r.db.WithContext(ctx).
				Table("product_units").
				Select("name").
				Where("id = ?", pData.ProductUnitID).
				Take(&unitName).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return "", nil, nil, err
				}
			}
			if unitName != "" {
				// Try to find unit with same name in target store
				var existingUnitID string
				if err := r.db.WithContext(ctx).
					Table("product_units").
					Select("id").
					Where("store_id = ? AND name = ?", targetStoreID, unitName).
					Take(&existingUnitID).Error; err != nil {
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						return "", nil, nil, err
					}
				}
				if existingUnitID != "" {
					destUnitID = existingUnitID
				} else {
					// No matching unit — find any active unit in target store
					var anyUnitID string
					if err := r.db.WithContext(ctx).
						Table("product_units").
						Select("id").
						Where("store_id = ? AND is_active = ?", targetStoreID, true).
						Limit(1).
						Take(&anyUnitID).Error; err != nil {
						if !errors.Is(err, gorm.ErrRecordNotFound) {
							return "", nil, nil, err
						}
					}
					if anyUnitID != "" {
						destUnitID = anyUnitID
					} else {
						// No units at all in target store — create a default one
						destUnitID = newID()
						if err := r.db.WithContext(ctx).Table("product_units").Create(map[string]any{
							"id":         destUnitID,
							"store_id":   targetStoreID,
							"name":       "ชิ้น",
							"is_active":  true,
							"created_at": now,
							"updated_at": now,
						}).Error; err != nil {
							return "", nil, nil, err
						}
					}
				}
			}
		} else {
			// Source product has no unit — find any active unit in target store
			var anyUnitID string
			if err := r.db.WithContext(ctx).
				Table("product_units").
				Select("id").
				Where("store_id = ? AND is_active = ?", targetStoreID, true).
				Limit(1).
				Take(&anyUnitID).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return "", nil, nil, err
				}
			}
			if anyUnitID != "" {
				destUnitID = anyUnitID
			} else {
				// Create a default unit
				destUnitID = newID()
				if err := r.db.WithContext(ctx).Table("product_units").Create(map[string]any{
					"id":         destUnitID,
					"store_id":   targetStoreID,
					"name":       "ชิ้น",
					"is_active":  true,
					"created_at": now,
					"updated_at": now,
				}).Error; err != nil {
					return "", nil, nil, err
				}
			}
		}

		// Create cloned product
		destPID := newID()
		if err := r.db.WithContext(ctx).Table("products").Create(map[string]any{
			"id":                     destPID,
			"store_id":               targetStoreID,
			"name":                   pData.Name,
			"sku":                    nilEmpty(pData.SKU),
			"barcode":                nilEmpty(pData.Barcode),
			"image_url":              nilEmpty(pData.ImageURL),
			"product_type_id":        nil,
			"product_unit_id":        destUnitID,
			"brand_id":               nil,
			"base_price":             pData.BasePrice,
			"cost_price":             pData.CostPrice,
			"special_price":          pData.SpecialPrice,
			"special_price_start_at": pData.SpecialPriceStartAt,
			"special_price_end_at":   pData.SpecialPriceEndAt,
			"is_active":              pData.IsActive,
			"min_stock":              pData.MinStock,
			"max_stock":              pData.MaxStock,
			"product_code":           nilEmpty(pData.ProductCode),
			"description":            nilEmpty(pData.Description),
			"storage_location":       nilEmpty(pData.StorageLocation),
			"created_at":             now,
			"updated_at":             now,
		}).Error; err != nil {
			return "", nil, nil, err
		}
		productIDMap[srcPID] = destPID
	}

	return newWarehouseID, locationIDMap, productIDMap, nil
}
