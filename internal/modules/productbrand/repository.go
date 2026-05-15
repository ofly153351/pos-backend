package productbrand

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, item ProductBrand) (ProductBrand, error)
	ListByStore(ctx context.Context, storeID string) ([]ProductBrand, error)
	GetByID(ctx context.Context, storeID, id string) (ProductBrand, error)
	Update(ctx context.Context, item ProductBrand) (ProductBrand, error)
	Delete(ctx context.Context, storeID, id string) error
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) PostgresRepository { return PostgresRepository{db: db} }

func (r PostgresRepository) Create(ctx context.Context, item ProductBrand) (ProductBrand, error) {
	payload := map[string]any{
		"id":         item.ID,
		"store_id":   item.StoreID,
		"name":       item.Name,
		"is_active":  item.IsActive,
		"created_at": item.CreatedAt,
		"updated_at": item.CreatedAt,
	}
	err := r.db.WithContext(ctx).Table("product_brands").Create(payload).Error
	if err != nil {
		return ProductBrand{}, err
	}
	item.UpdatedAt = item.CreatedAt
	return item, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]ProductBrand, error) {
	var items []ProductBrand
	err := r.db.WithContext(ctx).
		Model(&ProductBrand{}).
		Where("store_id = ?", storeID).
		Order("name ASC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, id string) (ProductBrand, error) {
	var item ProductBrand
	err := r.db.WithContext(ctx).
		Model(&ProductBrand{}).
		Where("store_id = ? AND id = ?", storeID, id).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductBrand{}, ErrProductBrandNotFound
		}
		return ProductBrand{}, err
	}
	return item, nil
}

func (r PostgresRepository) Update(ctx context.Context, item ProductBrand) (ProductBrand, error) {
	result := r.db.WithContext(ctx).
		Model(&ProductBrand{}).
		Where("store_id = ? AND id = ?", item.StoreID, item.ID).
		Updates(map[string]any{
			"name":       item.Name,
			"is_active":  item.IsActive,
			"updated_at": item.UpdatedAt,
		})
	if result.Error != nil {
		return ProductBrand{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ProductBrand{}, ErrProductBrandNotFound
	}
	return item, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, id string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, id).
		Delete(&ProductBrand{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductBrandNotFound
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
