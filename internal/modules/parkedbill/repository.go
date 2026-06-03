package parkedbill

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type productSnapshot struct {
	ID        string
	Name      string
	SKU       string
	BasePrice float64
}

type Repository interface {
	CreateParkedBill(ctx context.Context, bill *ParkedBill) error
	CreateParkedBillItems(ctx context.Context, items []ParkedBillItem) error
	ListByStore(ctx context.Context, storeID string) ([]ParkedBill, error)
	GetByID(ctx context.Context, billID string) (ParkedBill, error)
	GetItemsByBillID(ctx context.Context, billID string) ([]ParkedBillItem, error)
	Delete(ctx context.Context, billID string) error
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
	GetProductByID(ctx context.Context, productID string) (productSnapshot, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) CreateParkedBill(ctx context.Context, bill *ParkedBill) error {
	payload := map[string]any{
		"id":                       bill.ID,
		"store_id":                 bill.StoreID,
		"cashier_user_id":          bill.CashierUserID,
		"label":                    bill.Label,
		"bill_discount_amount":     bill.BillDiscountAmount,
		"bill_discount_type":       bill.BillDiscountType,
		"bill_discount_percent":    bill.BillDiscountPercent,
		"customer_settlement_mode": bill.CustomerSettlementMode,
		"payment_method":           bill.PaymentMethod,
		"vat_included":             bill.VATIncluded,
		"vat_percent":              bill.VATPercent,
		"created_at":               bill.CreatedAt,
	}
	payload["note"] = bill.Note
	if bill.CustomerID == "" {
		payload["customer_id"] = nil
	} else {
		payload["customer_id"] = bill.CustomerID
	}
	return r.db.WithContext(ctx).Table("parked_bills").Create(payload).Error
}

func (r PostgresRepository) CreateParkedBillItems(ctx context.Context, items []ParkedBillItem) error {
	for _, item := range items {
		itemPayload := map[string]any{
			"id":             item.ID,
			"parked_bill_id": item.ParkedBillID,
			"product_id":     item.ProductID,
			"product_name":   item.ProductName,
			"product_sku":    item.ProductSKU,
			"price":          item.Price,
			"quantity":       item.Quantity,
		}
		if item.DiscountType == "" {
			itemPayload["discount_type"] = nil
		} else {
			itemPayload["discount_type"] = item.DiscountType
		}
		if item.DiscountValue == nil {
			itemPayload["discount_value"] = nil
		} else {
			itemPayload["discount_value"] = *item.DiscountValue
		}
		if err := r.db.WithContext(ctx).Table("parked_bill_items").Create(itemPayload).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]ParkedBill, error) {
	var bills []ParkedBill
	err := r.db.WithContext(ctx).
		Model(&ParkedBill{}).
		Where("store_id = ?", storeID).
		Preload("Items").
		Order("created_at DESC").
		Find(&bills).Error
	return bills, err
}

func (r PostgresRepository) GetByID(ctx context.Context, billID string) (ParkedBill, error) {
	var bill ParkedBill
	err := r.db.WithContext(ctx).
		Model(&ParkedBill{}).
		Where("id = ?", billID).
		Preload("Items").
		First(&bill).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ParkedBill{}, ErrParkedBillNotFound
		}
		return ParkedBill{}, err
	}
	return bill, nil
}

func (r PostgresRepository) GetItemsByBillID(ctx context.Context, billID string) ([]ParkedBillItem, error) {
	var items []ParkedBillItem
	err := r.db.WithContext(ctx).
		Model(&ParkedBillItem{}).
		Where("parked_bill_id = ?", billID).
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) Delete(ctx context.Context, billID string) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	result := tx.Where("parked_bill_id = ?", billID).Delete(&ParkedBillItem{})
	if result.Error != nil {
		return result.Error
	}

	result = tx.Where("id = ?", billID).Delete(&ParkedBill{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrParkedBillNotFound
	}

	return tx.Commit().Error
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

func (r PostgresRepository) GetProductByID(ctx context.Context, productID string) (productSnapshot, error) {
	var snap productSnapshot
	err := r.db.WithContext(ctx).
		Table("products").
		Select("id, name, sku, base_price").
		Where("id = ?", productID).
		Take(&snap).Error
	return snap, err
}
