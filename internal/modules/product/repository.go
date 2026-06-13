package product

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, product Product) (Product, error)
	ListByStore(ctx context.Context, storeID string, page, limit int, stockStatus, sortBy string) ([]Product, int64, error)
	GetByID(ctx context.Context, storeID, productID string) (Product, error)
	Update(ctx context.Context, product Product) (Product, error)
	UpdateSKU(ctx context.Context, storeID, productID, sku string, updatedAt time.Time) error
	SKUExists(ctx context.Context, storeID, sku string) (bool, error)
	BarcodeExists(ctx context.Context, storeID, barcode string) (bool, error)
	ListProductIDsWithoutSKU(ctx context.Context, storeID string) ([]string, error)
	Delete(ctx context.Context, storeID, productID string) error
	SoftDelete(ctx context.Context, storeID, productID string) error
	ExistsByID(ctx context.Context, storeID, productID string) (bool, error)
	ProductTypeExists(ctx context.Context, storeID, productTypeID string) (bool, error)
	ProductUnitExists(ctx context.Context, storeID, productUnitID string) (bool, error)
	BrandExists(ctx context.Context, storeID, brandID string) (bool, error)
	LocationBelongsToStore(ctx context.Context, storeID, locationID string) (bool, error)
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

type productQueryRow struct {
	ID                  string     `gorm:"column:id"`
	StoreID             string     `gorm:"column:store_id"`
	ProductTypeID       *string    `gorm:"column:product_type_id"`
	ProductTypeName     *string    `gorm:"column:product_type_name"`
	ProductUnitID       *string    `gorm:"column:product_unit_id"`
	ProductUnitName     *string    `gorm:"column:product_unit_name"`
	BrandID             *string    `gorm:"column:brand_id"`
	BrandName           *string    `gorm:"column:brand_name"`
	Name                string     `gorm:"column:name"`
	SKU                 *string    `gorm:"column:sku"`
	Barcode             *string    `gorm:"column:barcode"`
	ProductCode         *string    `gorm:"column:product_code"`
	Description         *string    `gorm:"column:description"`
	StorageLocation     *string    `gorm:"column:storage_location"`
	DefaultLocationID   *string    `gorm:"column:default_location_id"`
	CostPrice           float64    `gorm:"column:cost_price"`
	ImageURL            *string    `gorm:"column:image_url"`
	MinStock            int        `gorm:"column:min_stock"`
	MaxStock            *int       `gorm:"column:max_stock"`
	BasePrice           float64    `gorm:"column:base_price"`
	SpecialPrice        *float64   `gorm:"column:special_price"`
	SpecialPriceStartAt *time.Time `gorm:"column:special_price_start_at"`
	SpecialPriceEndAt   *time.Time `gorm:"column:special_price_end_at"`
	TotalStock          int        `gorm:"column:total_stock"`
	StockStatus         string     `gorm:"column:stock_status"`
	IsActive            bool       `gorm:"column:is_active"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at"`
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, product Product) (Product, error) {
	payload := map[string]any{
		"id":                     product.ID,
		"store_id":               product.StoreID,
		"name":                   product.Name,
		"brand_id":               product.BrandID,
		"product_unit_id":        product.ProductUnitID,
		"min_stock":              product.MinStock,
		"max_stock":              product.MaxStock,
		"base_price":             product.BasePrice,
		"cost_price":             product.CostPrice,
		"special_price":          product.SpecialPrice,
		"special_price_start_at": product.SpecialPriceStartAt,
		"special_price_end_at":   product.SpecialPriceEndAt,
		"is_active":              product.IsActive,
		"created_at":             product.CreatedAt,
		"updated_at":             product.CreatedAt,
	}
	if product.ProductTypeID == "" {
		payload["product_type_id"] = nil
	} else {
		payload["product_type_id"] = product.ProductTypeID
	}
	if product.BrandID == "" {
		payload["brand_id"] = nil
	} else {
		payload["brand_id"] = product.BrandID
	}
	if product.SKU == "" {
		payload["sku"] = nil
	} else {
		payload["sku"] = product.SKU
	}
	if product.Barcode == "" {
		payload["barcode"] = nil
	} else {
		payload["barcode"] = product.Barcode
	}
	if product.ImageURL == "" {
		payload["image_url"] = nil
	} else {
		payload["image_url"] = product.ImageURL
	}
	if product.ProductCode == "" {
		payload["product_code"] = nil
	} else {
		payload["product_code"] = product.ProductCode
	}
	if product.Description == "" {
		payload["description"] = nil
	} else {
		payload["description"] = product.Description
	}
	if product.StorageLocation == "" {
		payload["storage_location"] = nil
	} else {
		payload["storage_location"] = product.StorageLocation
	}
	if product.DefaultLocationID == nil || *product.DefaultLocationID == "" {
		payload["default_location_id"] = nil
	} else {
		payload["default_location_id"] = *product.DefaultLocationID
	}

	if err := r.db.WithContext(ctx).Table("products").Create(payload).Error; err != nil {
		return Product{}, err
	}

	return r.GetByID(ctx, product.StoreID, product.ID)
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string, page, limit int, stockStatus, sortBy string) ([]Product, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	baseQuery := r.db.WithContext(ctx).
		Table("product_view pv").
		Where("pv.store_id = ?", storeID)

	if stockStatus == "out_of_stock" {
		baseQuery = baseQuery.Where("pv.total_stock = 0")
	} else if stockStatus == "low_stock" {
		baseQuery = baseQuery.Where("pv.total_stock > 0 AND pv.total_stock <= pv.min_stock")
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []productQueryRow
	err := baseQuery.
		Select(`
			pv.id, pv.store_id, pv.product_type_id, pv.product_type_name, pv.product_unit_id, pv.product_unit_name, pv.brand_id, pv.brand_name, pv.name, pv.sku, pv.barcode, pv.image_url, pv.min_stock, pv.max_stock, pv.base_price, pv.cost_price, pv.special_price, pv.special_price_start_at, pv.special_price_end_at, pv.total_stock,
			CASE
				WHEN pv.total_stock = 0 THEN 'out_of_stock'
				WHEN pv.total_stock > 0 AND pv.total_stock <= pv.min_stock THEN 'low_stock'
				ELSE 'active'
			END AS stock_status,
			pv.is_active, pv.created_at, pv.updated_at, pv.product_code, pv.description, pv.storage_location, pv.default_location_id
		`).
		Order(func() string {
			if sortBy == "updated_at" { return "pv.updated_at DESC" }
			return "pv.created_at DESC"
		}()).
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	now := time.Now().UTC()
	products := make([]Product, 0, len(rows))
	for _, row := range rows {
		item := row.toProduct()
		item.EffectivePrice = resolveEffectivePrice(item, now)
		products = append(products, item)
	}
	return products, total, nil
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, productID string) (Product, error) {
	var row productQueryRow
	err := r.db.WithContext(ctx).
		Table("product_view pv").
		Select(`
			pv.id, pv.store_id, pv.product_type_id, pv.product_type_name, pv.product_unit_id, pv.product_unit_name, pv.brand_id, pv.brand_name, pv.name, pv.sku, pv.barcode, pv.image_url, pv.min_stock, pv.max_stock, pv.base_price, pv.cost_price, pv.special_price, pv.special_price_start_at, pv.special_price_end_at, pv.total_stock,
			CASE
				WHEN pv.total_stock = 0 THEN 'out_of_stock'
				WHEN pv.total_stock > 0 AND pv.total_stock <= pv.min_stock THEN 'low_stock'
				ELSE 'active'
			END AS stock_status,
			pv.is_active, pv.created_at, pv.updated_at, pv.product_code, pv.description, pv.storage_location, pv.default_location_id
		`).
		Where("pv.store_id = ? AND pv.id = ?", storeID, productID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Product{}, ErrProductNotFound
		}
		return Product{}, err
	}

	product := row.toProduct()
	product.EffectivePrice = resolveEffectivePrice(product, time.Now().UTC())
	return product, nil
}

func (r PostgresRepository) Update(ctx context.Context, product Product) (Product, error) {
	updates := map[string]any{
		"name":                   product.Name,
		"brand_id":               product.BrandID,
		"product_unit_id":        product.ProductUnitID,
		"min_stock":              product.MinStock,
		"max_stock":              product.MaxStock,
		"base_price":             product.BasePrice,
		"cost_price":             product.CostPrice,
		"special_price":          product.SpecialPrice,
		"special_price_start_at": product.SpecialPriceStartAt,
		"special_price_end_at":   product.SpecialPriceEndAt,
		"is_active":              product.IsActive,
		"updated_at":             product.UpdatedAt,
	}
	if product.ProductTypeID == "" {
		updates["product_type_id"] = nil
	} else {
		updates["product_type_id"] = product.ProductTypeID
	}
	if product.BrandID == "" {
		updates["brand_id"] = nil
	} else {
		updates["brand_id"] = product.BrandID
	}
	if product.SKU == "" {
		updates["sku"] = nil
	} else {
		updates["sku"] = product.SKU
	}
	if product.Barcode == "" {
		updates["barcode"] = nil
	} else {
		updates["barcode"] = product.Barcode
	}
	if product.ImageURL == "" {
		updates["image_url"] = nil
	} else {
		updates["image_url"] = product.ImageURL
	}
	if product.ProductCode == "" {
		updates["product_code"] = nil
	} else {
		updates["product_code"] = product.ProductCode
	}
	if product.Description == "" {
		updates["description"] = nil
	} else {
		updates["description"] = product.Description
	}
	if product.StorageLocation == "" {
		updates["storage_location"] = nil
	} else {
		updates["storage_location"] = product.StorageLocation
	}
	if product.DefaultLocationID == nil || *product.DefaultLocationID == "" {
		updates["default_location_id"] = nil
	} else {
		updates["default_location_id"] = *product.DefaultLocationID
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Model(&Product{}).
			Where("store_id = ? AND id = ?", product.StoreID, product.ID).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrProductNotFound
		}
		return nil
	}); err != nil {
		return Product{}, err
	}

	return r.GetByID(ctx, product.StoreID, product.ID)
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, productID string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, productID).
		Delete(&Product{})
	if result.Error != nil {
		if isForeignKeyViolation(result.Error) {
			return ErrProductInUse
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r PostgresRepository) ExistsByID(ctx context.Context, storeID, productID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Product{}).
		Where("store_id = ? AND id = ?", storeID, productID).
		Count(&count).Error
	return count > 0, err
}

func (r PostgresRepository) SoftDelete(ctx context.Context, storeID, productID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Zero all stock quantities
		if err := tx.Exec(`UPDATE stocks SET quantity = 0, updated_at = NOW() WHERE product_id = ?`, productID).Error; err != nil {
			return err
		}
		// 2. Delete warehouse inventory
		if err := tx.Exec(`DELETE FROM warehouse_inventory WHERE product_id = ?`, productID).Error; err != nil {
			return err
		}
		// 3. Delete stock movements for this product
		if err := tx.Exec(`DELETE FROM stock_movements WHERE product_id = ?`, productID).Error; err != nil {
			return err
		}
		// 4. Set product inactive (soft delete)
		if err := tx.Model(&Product{}).
			Where("store_id = ? AND id = ?", storeID, productID).
			Update("is_active", false).
			Error; err != nil {
			return err
		}
		return nil
	})
}

func (r PostgresRepository) UpdateSKU(ctx context.Context, storeID, productID, sku string, updatedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&Product{}).
		Where("store_id = ? AND id = ?", storeID, productID).
		Updates(map[string]any{
			"sku":        sku,
			"updated_at": updatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r PostgresRepository) SKUExists(ctx context.Context, storeID, sku string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("products").
		Where("store_id = ? AND sku = ?", storeID, sku).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r PostgresRepository) BarcodeExists(ctx context.Context, storeID, barcode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("products").
		Where("store_id = ? AND barcode = ?", storeID, barcode).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r PostgresRepository) ListProductIDsWithoutSKU(ctx context.Context, storeID string) ([]string, error) {
	type row struct {
		ID string `gorm:"column:id"`
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("products").
		Select("id").
		Where("store_id = ? AND (sku IS NULL OR TRIM(sku) = '')", storeID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, item := range rows {
		ids = append(ids, item.ID)
	}
	return ids, nil
}

func (r PostgresRepository) ProductTypeExists(ctx context.Context, storeID, productTypeID string) (bool, error) {
	if productTypeID == "" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("product_types").
		Where("store_id = ? AND id = ?", storeID, productTypeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r PostgresRepository) ProductUnitExists(ctx context.Context, storeID, productUnitID string) (bool, error) {
	if productUnitID == "" {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("product_units").
		Where("store_id = ? AND id = ?", storeID, productUnitID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r PostgresRepository) BrandExists(ctx context.Context, storeID, brandID string) (bool, error) {
	if brandID == "" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("product_brands").
		Where("store_id = ? AND id = ?", storeID, brandID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r PostgresRepository) LocationBelongsToStore(ctx context.Context, storeID, locationID string) (bool, error) {
	if locationID == "" {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("locations").
		Where("store_id = ? AND id = ?", storeID, locationID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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

func (row productQueryRow) toProduct() Product {
	product := Product{
		ID:                  row.ID,
		StoreID:             row.StoreID,
		Name:                row.Name,
		MinStock:            row.MinStock,
		MaxStock:            row.MaxStock,
		BasePrice:           row.BasePrice,
		SpecialPrice:        row.SpecialPrice,
		SpecialPriceStartAt: row.SpecialPriceStartAt,
		SpecialPriceEndAt:   row.SpecialPriceEndAt,
		IsActive:            row.IsActive,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
		CostPrice:           row.CostPrice,
		TotalStock:          row.TotalStock,
		StockStatus:         row.StockStatus,
	}
	if row.BrandID != nil {
		product.BrandID = *row.BrandID
	}
	if row.BrandName != nil {
		product.BrandName = *row.BrandName
	}
	if row.ProductTypeID != nil {
		product.ProductTypeID = *row.ProductTypeID
	}
	if row.ProductTypeName != nil {
		product.ProductTypeName = *row.ProductTypeName
	}
	if row.ProductUnitID != nil {
		product.ProductUnitID = *row.ProductUnitID
	}
	if row.ProductUnitName != nil {
		product.ProductUnitName = *row.ProductUnitName
	}
	if row.SKU != nil {
		product.SKU = *row.SKU
	}
	if row.Barcode != nil {
		product.Barcode = *row.Barcode
	}
	if row.ImageURL != nil {
		product.ImageURL = *row.ImageURL
	}
	if row.ProductCode != nil {
		product.ProductCode = *row.ProductCode
	}
	if row.Description != nil {
		product.Description = *row.Description
	}
	if row.StorageLocation != nil {
		product.StorageLocation = *row.StorageLocation
	}
	product.DefaultLocationID = row.DefaultLocationID
	return product
}

func isForeignKeyViolation(err error) bool {
	return strings.Contains(err.Error(), "SQLSTATE 23503")
}
