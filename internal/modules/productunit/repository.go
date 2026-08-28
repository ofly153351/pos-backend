package productunit

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, unit ProductUnit) (ProductUnit, error)
	ListByStore(ctx context.Context, storeID string) ([]ProductUnit, error)
	GetByID(ctx context.Context, storeID, id string) (ProductUnit, error)
	Update(ctx context.Context, unit ProductUnit) (ProductUnit, error)
	Delete(ctx context.Context, storeID, id string) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository { return PostgresRepository{db: db} }

func (r PostgresRepository) Create(ctx context.Context, unit ProductUnit) (ProductUnit, error) {
	payload := map[string]any{
		"id":         unit.ID,
		"store_id":   unit.StoreID,
		"name":       unit.Name,
		"is_active":  unit.IsActive,
		"created_at": unit.CreatedAt,
		"updated_at": unit.CreatedAt,
	}
	if unit.Description == "" {
		payload["description"] = nil
	} else {
		payload["description"] = unit.Description
	}
	err := r.db.WithContext(ctx).Table("product_units").Create(payload).Error
	if err != nil {
		return ProductUnit{}, err
	}
	unit.UpdatedAt = unit.CreatedAt
	return unit, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]ProductUnit, error) {
	var list []ProductUnit
	err := r.db.WithContext(ctx).
		Model(&ProductUnit{}).
		Select("product_units.*, COALESCE((SELECT COUNT(*) FROM products p WHERE p.product_unit_id = product_units.id), 0) AS product_count").
		Where("product_units.store_id = ?", storeID).
		Order("product_units.name ASC").
		Find(&list).Error
	return list, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, id string) (ProductUnit, error) {
	var unit ProductUnit
	err := r.db.WithContext(ctx).
		Model(&ProductUnit{}).
		Where("store_id = ? AND id = ?", storeID, id).
		First(&unit).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductUnit{}, ErrProductUnitNotFound
		}
		return ProductUnit{}, err
	}
	return unit, nil
}

func (r PostgresRepository) Update(ctx context.Context, unit ProductUnit) (ProductUnit, error) {
	updates := map[string]any{
		"name":       unit.Name,
		"is_active":  unit.IsActive,
		"updated_at": unit.UpdatedAt,
	}
	if unit.Description == "" {
		updates["description"] = nil
	} else {
		updates["description"] = unit.Description
	}
	result := r.db.WithContext(ctx).
		Model(&ProductUnit{}).
		Where("store_id = ? AND id = ?", unit.StoreID, unit.ID).
		Updates(updates)
	if result.Error != nil {
		return ProductUnit{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ProductUnit{}, ErrProductUnitNotFound
	}
	return unit, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, id string) error {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("products").
		Where("store_id = ? AND product_unit_id = ?", storeID, id).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrProductUnitInUse
	}

	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, id).
		Delete(&ProductUnit{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductUnitNotFound
	}
	return nil
}
