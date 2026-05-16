package warehouse

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, item Warehouse) (Warehouse, error)
	ListByStore(ctx context.Context, storeID string) ([]Warehouse, error)
	GetByID(ctx context.Context, storeID, id string) (Warehouse, error)
	Update(ctx context.Context, item Warehouse) (Warehouse, error)
	Delete(ctx context.Context, storeID, id string) error
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)

	// Warehouse-Product association
	AddProduct(ctx context.Context, wp WarehouseProduct) (WarehouseProduct, error)
	ListProducts(ctx context.Context, warehouseID string) ([]WarehouseProduct, error)
	UpdateProduct(ctx context.Context, warehouseID, productID string, quantity int) error
	RemoveProduct(ctx context.Context, warehouseID, productID string) error
	ProductExistsInWarehouse(ctx context.Context, warehouseID, productID string) (bool, error)
	ProductBelongsToStore(ctx context.Context, storeID, productID string) (bool, error)
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

// Warehouse-Product repository methods

func (r PostgresRepository) AddProduct(ctx context.Context, wp WarehouseProduct) (WarehouseProduct, error) {
	payload := map[string]any{
		"id":           wp.ID,
		"warehouse_id": wp.WarehouseID,
		"product_id":   wp.ProductID,
		"quantity":     wp.Quantity,
		"created_at":   wp.CreatedAt,
	}
	err := r.db.WithContext(ctx).Table("warehouse_products").Create(payload).Error
	if err != nil {
		return WarehouseProduct{}, err
	}
	return wp, nil
}

func (r PostgresRepository) ListProducts(ctx context.Context, warehouseID string) ([]WarehouseProduct, error) {
	var items []WarehouseProduct
	err := r.db.WithContext(ctx).
		Table("warehouse_products").
		Select(`
			warehouse_products.id,
			warehouse_products.warehouse_id,
			warehouse_products.product_id,
			warehouse_products.quantity,
			warehouse_products.created_at,
			pv.name AS product_name,
			pv.sku AS product_sku,
			pv.barcode AS product_barcode,
			pv.base_price AS product_price,
			pv.image_url,
			pv.product_type_name,
			pv.product_unit_name,
			pv.min_stock AS product_min_stock,
			pv.max_stock AS product_max_stock,
			pv.quantity AS product_quantity
		`).
		Joins("JOIN product_view pv ON pv.id = warehouse_products.product_id").
		Where("warehouse_products.warehouse_id = ?", warehouseID).
		Order("pv.name ASC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) UpdateProduct(ctx context.Context, warehouseID, productID string, quantity int) error {
	result := r.db.WithContext(ctx).
		Table("warehouse_products").
		Where("warehouse_id = ? AND product_id = ?", warehouseID, productID).
		Update("quantity", quantity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductNotInWarehouse
	}
	return nil
}

func (r PostgresRepository) RemoveProduct(ctx context.Context, warehouseID, productID string) error {
	result := r.db.WithContext(ctx).
		Table("warehouse_products").
		Where("warehouse_id = ? AND product_id = ?", warehouseID, productID).
		Delete(nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductNotInWarehouse
	}
	return nil
}

func (r PostgresRepository) ProductExistsInWarehouse(ctx context.Context, warehouseID, productID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("warehouse_products").
		Where("warehouse_id = ? AND product_id = ?", warehouseID, productID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
