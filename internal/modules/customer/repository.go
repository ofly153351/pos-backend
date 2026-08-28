package customer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, customer Customer) (Customer, error)
	ListByStore(ctx context.Context, storeID string) ([]CustomerListItem, error)
	GetByID(ctx context.Context, storeID, customerID string) (Customer, error)
	Update(ctx context.Context, customer Customer) (Customer, error)
	Delete(ctx context.Context, storeID, customerID string) error
	GetNetworkLevel(ctx context.Context, storeID, customerID string) (int, error)
	GetLevelDiscountPercent(ctx context.Context, storeID string, level int) (float64, error)
	ListLevelDiscounts(ctx context.Context, storeID string) ([]LevelDiscount, error)
	UpsertLevelDiscount(ctx context.Context, item LevelDiscount) (LevelDiscount, error)
	DeleteLevelDiscount(ctx context.Context, storeID string, level int) error
	// Shipping addresses
	ListShippingAddresses(ctx context.Context, customerID string) ([]CustomerShippingAddress, error)
	CreateShippingAddress(ctx context.Context, addr CustomerShippingAddress) (CustomerShippingAddress, error)
	UpdateShippingAddress(ctx context.Context, addr CustomerShippingAddress) (CustomerShippingAddress, error)
	DeleteShippingAddress(ctx context.Context, addrID, customerID string) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

// setOrNull writes a trimmed value to the column map, or SQL NULL when the value is
// blank — preserving the customers table's empty->NULL convention so optional text
// columns round-trip cleanly instead of storing empty strings.
func setOrNull(m map[string]any, col, val string) {
	if strings.TrimSpace(val) == "" {
		m[col] = nil
	} else {
		m[col] = strings.TrimSpace(val)
	}
}

func (r PostgresRepository) Create(ctx context.Context, customer Customer) (Customer, error) {
	// Auto-generate a per-store member code ("M00001", ...) when the caller did not
	// supply one. The unique (store_id, member_code) index guards against the rare
	// concurrent-count collision.
	if strings.TrimSpace(customer.MemberCode) == "" {
		code, err := r.nextMemberCode(ctx, customer.StoreID)
		if err != nil {
			return Customer{}, err
		}
		customer.MemberCode = code
	}

	payload := map[string]any{
		"id":             customer.ID,
		"store_id":       customer.StoreID,
		"customer_level": customer.Level,
		"member_code":    customer.MemberCode,
		"points":         customer.Points,
		"full_name":      customer.FullName,
		"is_active":      customer.IsActive,
		"created_at":     customer.CreatedAt,
		"updated_at":     customer.CreatedAt,
	}
	setOrNull(payload, "phone", customer.Phone)
	setOrNull(payload, "email", customer.Email)
	setOrNull(payload, "address", customer.Address)
	setOrNull(payload, "note", customer.Note)
	setOrNull(payload, "tax_id", customer.TaxID)
	setOrNull(payload, "branch", customer.Branch)
	setOrNull(payload, "shipping_contact", customer.ShippingContact)
	setOrNull(payload, "shipping_phone", customer.ShippingPhone)
	setOrNull(payload, "shipping_address", customer.ShippingAddress)
	setOrNull(payload, "shipping_province", customer.ShippingProvince)
	setOrNull(payload, "shipping_district", customer.ShippingDistrict)
	setOrNull(payload, "shipping_postal_code", customer.ShippingPostalCode)
	setOrNull(payload, "delivery_note", customer.DeliveryNote)

	if err := r.db.WithContext(ctx).Table("customers").Create(payload).Error; err != nil {
		return Customer{}, err
	}
	customer.UpdatedAt = customer.CreatedAt
	return customer, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]CustomerListItem, error) {
	// total_purchase / total_bills come from completed sales aggregated per customer.
	// LEFT JOIN a pre-grouped subquery so customers with no sales still return (0/0)
	// instead of being dropped. Voided sales and loans (ยืมสินค้า — borrowed goods, not
	// revenue; a returned loan's sale row stays 'completed') are excluded so a customer's
	// displayed lifetime spend/bill count reflects only real purchases incl. credit sales.
	var items []CustomerListItem
	err := r.db.WithContext(ctx).
		Table("customers AS c").
		Select(`c.*,
			COALESCE(s.total_purchase, 0) AS total_purchase,
			COALESCE(s.total_bills, 0)    AS total_bills`).
		Joins(`LEFT JOIN (
			SELECT customer_id,
			       SUM(total_amount) AS total_purchase,
			       COUNT(*)          AS total_bills
			FROM sales
			WHERE store_id = ? AND customer_id IS NOT NULL
			  AND status <> 'voided'
			  AND NOT EXISTS (SELECT 1 FROM credit_sales cs WHERE cs.sale_id = sales.id AND cs.type = 'loan')
			GROUP BY customer_id
		) s ON s.customer_id = c.id`, storeID).
		Where("c.store_id = ?", storeID).
		Order("c.created_at DESC").
		Scan(&items).Error
	return items, err
}

// nextMemberCode returns the next per-store member code in the "M00001" series,
// matching the backfill format in migration 044.
func (r PostgresRepository) nextMemberCode(ctx context.Context, storeID string) (string, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&Customer{}).
		Where("store_id = ?", storeID).
		Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("M%05d", count+1), nil
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, customerID string) (Customer, error) {
	var item Customer
	err := r.db.WithContext(ctx).
		Model(&Customer{}).
		Where("store_id = ? AND id = ?", storeID, customerID).
		Preload("ShippingAddresses", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
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
	setOrNull(updates, "phone", customer.Phone)
	setOrNull(updates, "email", customer.Email)
	setOrNull(updates, "address", customer.Address)
	setOrNull(updates, "note", customer.Note)
	setOrNull(updates, "tax_id", customer.TaxID)
	setOrNull(updates, "branch", customer.Branch)
	setOrNull(updates, "shipping_contact", customer.ShippingContact)
	setOrNull(updates, "shipping_phone", customer.ShippingPhone)
	setOrNull(updates, "shipping_address", customer.ShippingAddress)
	setOrNull(updates, "shipping_province", customer.ShippingProvince)
	setOrNull(updates, "shipping_district", customer.ShippingDistrict)
	setOrNull(updates, "shipping_postal_code", customer.ShippingPostalCode)
	setOrNull(updates, "delivery_note", customer.DeliveryNote)

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

func (r PostgresRepository) ListShippingAddresses(ctx context.Context, customerID string) ([]CustomerShippingAddress, error) {
	var items []CustomerShippingAddress
	err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("created_at ASC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) CreateShippingAddress(ctx context.Context, addr CustomerShippingAddress) (CustomerShippingAddress, error) {
	if err := r.db.WithContext(ctx).Create(&addr).Error; err != nil {
		return CustomerShippingAddress{}, err
	}
	return addr, nil
}

func (r PostgresRepository) UpdateShippingAddress(ctx context.Context, addr CustomerShippingAddress) (CustomerShippingAddress, error) {
	result := r.db.WithContext(ctx).
		Model(&CustomerShippingAddress{}).
		Where("id = ? AND customer_id = ?", addr.ID, addr.CustomerID).
		Updates(map[string]any{
			"label":                addr.Label,
			"recipient_name":       addr.RecipientName,
			"recipient_phone":      addr.RecipientPhone,
			"address":              addr.Address,
			"sub_district":         addr.SubDistrict,
			"district":             addr.District,
			"province":             addr.Province,
			"postal_code":          addr.PostalCode,
			"note":                 addr.Note,
			"use_customer_address": addr.UseCustomerAddress,
			"is_default":           addr.IsDefault,
			"updated_at":           addr.UpdatedAt,
		})
	if result.Error != nil {
		return CustomerShippingAddress{}, result.Error
	}
	if result.RowsAffected == 0 {
		return CustomerShippingAddress{}, ErrShippingAddressNotFound
	}
	return addr, nil
}

func (r PostgresRepository) DeleteShippingAddress(ctx context.Context, addrID, customerID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND customer_id = ?", addrID, customerID).
		Delete(&CustomerShippingAddress{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrShippingAddressNotFound
	}
	return nil
}
