package producttype

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, item ProductType) (ProductType, error)
	ListByStore(ctx context.Context, storeID string) ([]ProductType, error)
	GetByID(ctx context.Context, storeID, id string) (ProductType, error)
	Update(ctx context.Context, item ProductType) (ProductType, error)
	Delete(ctx context.Context, storeID, id string) error
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) PostgresRepository { return PostgresRepository{db: db} }

func (r PostgresRepository) Create(ctx context.Context, item ProductType) (ProductType, error) {
	payload := map[string]any{
		"id":         item.ID,
		"store_id":   item.StoreID,
		"name":       item.Name,
		"is_active":  item.IsActive,
		"created_at": item.CreatedAt,
		"updated_at": item.CreatedAt,
	}
	if item.Description == "" {
		payload["description"] = nil
	} else {
		payload["description"] = item.Description
	}
	err := r.db.WithContext(ctx).Table("product_types").Create(payload).Error
	if err != nil {
		return ProductType{}, err
	}
	item.UpdatedAt = item.CreatedAt
	return item, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]ProductType, error) {
	var items []ProductType
	err := r.db.WithContext(ctx).
		Model(&ProductType{}).
		Where("store_id = ?", storeID).
		Order("name ASC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, id string) (ProductType, error) {
	var item ProductType
	err := r.db.WithContext(ctx).
		Model(&ProductType{}).
		Where("store_id = ? AND id = ?", storeID, id).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductType{}, ErrProductTypeNotFound
		}
		return ProductType{}, err
	}
	return item, nil
}

func (r PostgresRepository) Update(ctx context.Context, item ProductType) (ProductType, error) {
	updates := map[string]any{
		"name":       item.Name,
		"is_active":  item.IsActive,
		"updated_at": item.UpdatedAt,
	}
	if item.Description == "" {
		updates["description"] = nil
	} else {
		updates["description"] = item.Description
	}
	result := r.db.WithContext(ctx).
		Model(&ProductType{}).
		Where("store_id = ? AND id = ?", item.StoreID, item.ID).
		Updates(updates)
	if result.Error != nil {
		return ProductType{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ProductType{}, ErrProductTypeNotFound
	}
	return item, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, id string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, id).
		Delete(&ProductType{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductTypeNotFound
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
