package customer

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, customer Customer) (Customer, error)
	ListByStore(ctx context.Context, storeID string) ([]Customer, error)
	GetByID(ctx context.Context, storeID, customerID string) (Customer, error)
	Update(ctx context.Context, customer Customer) (Customer, error)
	Delete(ctx context.Context, storeID, customerID string) error
	GetNetworkLevel(ctx context.Context, storeID, customerID string) (int, error)
	GetLevelDiscountPercent(ctx context.Context, storeID string, level int) (float64, error)
	ListLevelDiscounts(ctx context.Context, storeID string) ([]LevelDiscount, error)
	UpsertLevelDiscount(ctx context.Context, item LevelDiscount) (LevelDiscount, error)
	DeleteLevelDiscount(ctx context.Context, storeID string, level int) error
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, customer Customer) (Customer, error) {
	payload := map[string]any{
		"id":             customer.ID,
		"store_id":       customer.StoreID,
		"customer_level": customer.Level,
		"full_name":      customer.FullName,
		"is_active":      customer.IsActive,
		"created_at":     customer.CreatedAt,
		"updated_at":     customer.CreatedAt,
	}
	if strings.TrimSpace(customer.Phone) == "" {
		payload["phone"] = nil
	} else {
		payload["phone"] = strings.TrimSpace(customer.Phone)
	}
	if strings.TrimSpace(customer.Email) == "" {
		payload["email"] = nil
	} else {
		payload["email"] = strings.TrimSpace(customer.Email)
	}
	if strings.TrimSpace(customer.Address) == "" {
		payload["address"] = nil
	} else {
		payload["address"] = strings.TrimSpace(customer.Address)
	}
	if strings.TrimSpace(customer.Note) == "" {
		payload["note"] = nil
	} else {
		payload["note"] = strings.TrimSpace(customer.Note)
	}
	if strings.TrimSpace(customer.TaxID) == "" {
		payload["tax_id"] = nil
	} else {
		payload["tax_id"] = strings.TrimSpace(customer.TaxID)
	}
	if strings.TrimSpace(customer.Branch) == "" {
		payload["branch"] = nil
	} else {
		payload["branch"] = strings.TrimSpace(customer.Branch)
	}

	if err := r.db.WithContext(ctx).Table("customers").Create(payload).Error; err != nil {
		return Customer{}, err
	}
	customer.UpdatedAt = customer.CreatedAt
	return customer, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Customer, error) {
	var items []Customer
	err := r.db.WithContext(ctx).
		Model(&Customer{}).
		Where("store_id = ?", storeID).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, customerID string) (Customer, error) {
	var item Customer
	err := r.db.WithContext(ctx).
		Model(&Customer{}).
		Where("store_id = ? AND id = ?", storeID, customerID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Customer{}, ErrCustomerNotFound
		}
		return Customer{}, err
	}
	return item, nil
}

func (r PostgresRepository) Update(ctx context.Context, customer Customer) (Customer, error) {
	updates := map[string]any{
		"customer_level": customer.Level,
		"full_name":      customer.FullName,
		"is_active":      customer.IsActive,
		"updated_at":     customer.UpdatedAt,
	}
	if strings.TrimSpace(customer.Phone) == "" {
		updates["phone"] = nil
	} else {
		updates["phone"] = strings.TrimSpace(customer.Phone)
	}
	if strings.TrimSpace(customer.Email) == "" {
		updates["email"] = nil
	} else {
		updates["email"] = strings.TrimSpace(customer.Email)
	}
	if strings.TrimSpace(customer.Address) == "" {
		updates["address"] = nil
	} else {
		updates["address"] = strings.TrimSpace(customer.Address)
	}
	if strings.TrimSpace(customer.Note) == "" {
		updates["note"] = nil
	} else {
		updates["note"] = strings.TrimSpace(customer.Note)
	}
	if strings.TrimSpace(customer.TaxID) == "" {
		updates["tax_id"] = nil
	} else {
		updates["tax_id"] = strings.TrimSpace(customer.TaxID)
	}
	if strings.TrimSpace(customer.Branch) == "" {
		updates["branch"] = nil
	} else {
		updates["branch"] = strings.TrimSpace(customer.Branch)
	}

	result := r.db.WithContext(ctx).
		Model(&Customer{}).
		Where("store_id = ? AND id = ?", customer.StoreID, customer.ID).
		Updates(updates)
	if result.Error != nil {
		return Customer{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Customer{}, ErrCustomerNotFound
	}
	return customer, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, customerID string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, customerID).
		Delete(&Customer{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCustomerNotFound
	}
	return nil
}

func (r PostgresRepository) GetNetworkLevel(ctx context.Context, storeID, customerID string) (int, error) {
	var item Customer
	err := r.db.WithContext(ctx).
		Model(&Customer{}).
		Select("customer_level").
		Where("store_id = ? AND id = ?", storeID, customerID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrCustomerNotFound
		}
		return 0, err
	}
	return item.Level, nil
}

func (r PostgresRepository) GetLevelDiscountPercent(ctx context.Context, storeID string, level int) (float64, error) {
	var item LevelDiscount
	err := r.db.WithContext(ctx).
		Model(&LevelDiscount{}).
		Where("store_id = ? AND level = ?", storeID, level).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return item.DiscountPercent, nil
}

func (r PostgresRepository) ListLevelDiscounts(ctx context.Context, storeID string) ([]LevelDiscount, error) {
	var items []LevelDiscount
	err := r.db.WithContext(ctx).
		Model(&LevelDiscount{}).
		Where("store_id = ?", storeID).
		Order("level ASC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) UpsertLevelDiscount(ctx context.Context, item LevelDiscount) (LevelDiscount, error) {
	item.UpdatedAt = item.CreatedAt
	err := r.db.WithContext(ctx).
		Model(&LevelDiscount{}).
		Where("store_id = ? AND level = ?", item.StoreID, item.Level).
		Assign(map[string]any{
			"discount_percent": item.DiscountPercent,
			"updated_at":       item.CreatedAt,
		}).
		FirstOrCreate(&item).Error
	if err != nil {
		return LevelDiscount{}, err
	}
	return item, nil
}

func (r PostgresRepository) DeleteLevelDiscount(ctx context.Context, storeID string, level int) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND level = ?", storeID, level).
		Delete(&LevelDiscount{})
	return result.Error
}

func (r PostgresRepository) UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager", "cashier"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
