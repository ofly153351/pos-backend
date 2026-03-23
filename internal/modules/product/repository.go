package product

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, product Product) (Product, error)
	ListByStore(ctx context.Context, storeID string) ([]Product, error)
	GetByID(ctx context.Context, storeID, productID string) (Product, error)
	Update(ctx context.Context, product Product) (Product, error)
	Delete(ctx context.Context, storeID, productID string) error
	ProductTypeExists(ctx context.Context, storeID, productTypeID string) (bool, error)
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
	Name                string     `gorm:"column:name"`
	SKU                 *string    `gorm:"column:sku"`
	UnitType            string     `gorm:"column:unit_type"`
	ImageURL            *string    `gorm:"column:image_url"`
	Quantity            int        `gorm:"column:quantity"`
	BasePrice           float64    `gorm:"column:base_price"`
	SpecialPrice        *float64   `gorm:"column:special_price"`
	SpecialPriceStartAt *time.Time `gorm:"column:special_price_start_at"`
	SpecialPriceEndAt   *time.Time `gorm:"column:special_price_end_at"`
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
		"unit_type":              product.UnitType,
		"quantity":               product.Quantity,
		"base_price":             product.BasePrice,
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
	if product.SKU == "" {
		payload["sku"] = nil
	} else {
		payload["sku"] = product.SKU
	}
	if product.ImageURL == "" {
		payload["image_url"] = nil
	} else {
		payload["image_url"] = product.ImageURL
	}

	if err := r.db.WithContext(ctx).Table("products").Create(payload).Error; err != nil {
		return Product{}, err
	}

	product.EffectivePrice = resolveEffectivePrice(product, time.Now().UTC())
	product.UpdatedAt = product.CreatedAt
	return product, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Product, error) {
	var rows []productQueryRow
	err := r.db.WithContext(ctx).
		Table("products AS p").
		Select("p.id, p.store_id, p.product_type_id, pt.name AS product_type_name, p.name, p.sku, p.unit_type, p.image_url, p.quantity, p.base_price, p.special_price, p.special_price_start_at, p.special_price_end_at, p.is_active, p.created_at, p.updated_at").
		Joins("LEFT JOIN product_types pt ON pt.id = p.product_type_id").
		Where("p.store_id = ?", storeID).
		Order("p.created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	products := make([]Product, 0, len(rows))
	for _, row := range rows {
		item := row.toProduct()
		item.EffectivePrice = resolveEffectivePrice(item, now)
		products = append(products, item)
	}
	return products, nil
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, productID string) (Product, error) {
	var row productQueryRow
	err := r.db.WithContext(ctx).
		Table("products AS p").
		Select("p.id, p.store_id, p.product_type_id, pt.name AS product_type_name, p.name, p.sku, p.unit_type, p.image_url, p.quantity, p.base_price, p.special_price, p.special_price_start_at, p.special_price_end_at, p.is_active, p.created_at, p.updated_at").
		Joins("LEFT JOIN product_types pt ON pt.id = p.product_type_id").
		Where("p.store_id = ? AND p.id = ?", storeID, productID).
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
		"unit_type":              product.UnitType,
		"quantity":               product.Quantity,
		"base_price":             product.BasePrice,
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
	if product.SKU == "" {
		updates["sku"] = nil
	} else {
		updates["sku"] = product.SKU
	}
	if product.ImageURL == "" {
		updates["image_url"] = nil
	} else {
		updates["image_url"] = product.ImageURL
	}

	result := r.db.WithContext(ctx).
		Model(&Product{}).
		Where("store_id = ? AND id = ?", product.StoreID, product.ID).
		Updates(updates)
	if result.Error != nil {
		return Product{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Product{}, ErrProductNotFound
	}

	product.EffectivePrice = resolveEffectivePrice(product, time.Now().UTC())
	return product, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, productID string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, productID).
		Delete(&Product{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}
	return nil
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
		UnitType:            row.UnitType,
		Quantity:            row.Quantity,
		BasePrice:           row.BasePrice,
		SpecialPrice:        row.SpecialPrice,
		SpecialPriceStartAt: row.SpecialPriceStartAt,
		SpecialPriceEndAt:   row.SpecialPriceEndAt,
		IsActive:            row.IsActive,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.ProductTypeID != nil {
		product.ProductTypeID = *row.ProductTypeID
	}
	if row.ProductTypeName != nil {
		product.ProductTypeName = *row.ProductTypeName
	}
	if row.SKU != nil {
		product.SKU = *row.SKU
	}
	if row.ImageURL != nil {
		product.ImageURL = *row.ImageURL
	}
	return product
}
