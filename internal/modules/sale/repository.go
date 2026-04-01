package sale

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, sale Sale) (Sale, error)
	ListByStore(ctx context.Context, storeID string) ([]Sale, error)
	GetByID(ctx context.Context, storeID, saleID string) (Sale, error)
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, sale Sale) (Sale, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return Sale{}, tx.Error
	}
	defer tx.Rollback()

	for index, item := range sale.Items {
		product, err := r.lockProductForSale(ctx, tx, sale.StoreID, item.ProductID)
		if err != nil {
			return Sale{}, err
		}
		if !product.IsActive {
			return Sale{}, ErrProductInactive
		}
		if product.Quantity < item.Quantity {
			return Sale{}, fmt.Errorf("%w for product %s", ErrInsufficientStock, item.ProductID)
		}

		unitPrice := resolveEffectivePrice(product, sale.SoldAt)
		manualDiscountPerUnit, err := calculateDiscount(item.DiscountType, item.DiscountValue, unitPrice)
		if err != nil {
			return Sale{}, err
		}
		networkDiscountPerUnit := calculateNetworkDiscount(sale.NetworkDiscountPercent, unitPrice, manualDiscountPerUnit)
		discountAmountPerUnit := manualDiscountPerUnit + networkDiscountPerUnit
		if discountAmountPerUnit > unitPrice {
			discountAmountPerUnit = unitPrice
		}

		sale.Items[index].ID = newID()
		sale.Items[index].SaleID = sale.ID
		sale.Items[index].ProductName = product.Name
		sale.Items[index].SKU = product.SKU
		sale.Items[index].UnitType = product.UnitType
		sale.Items[index].UnitPrice = unitPrice
		sale.Items[index].DiscountType = normalizeDiscountType(item.DiscountType)
		sale.Items[index].DiscountAmountPerUnit = discountAmountPerUnit
		sale.Items[index].LineSubtotal = unitPrice * float64(item.Quantity)
		sale.Items[index].LineDiscountTotal = discountAmountPerUnit * float64(item.Quantity)
		sale.Items[index].LineTotal = unitPrice * float64(item.Quantity)
		sale.Items[index].LineTotal -= sale.Items[index].LineDiscountTotal
		sale.Items[index].LineSubtotal = roundMoney(sale.Items[index].LineSubtotal)
		sale.Items[index].LineDiscountTotal = roundMoney(sale.Items[index].LineDiscountTotal)
		sale.Items[index].LineTotal = roundMoney(sale.Items[index].LineTotal)
		sale.Items[index].CreatedAt = sale.CreatedAt
		sale.SubtotalAmount += sale.Items[index].LineSubtotal
		sale.DiscountAmount += sale.Items[index].LineDiscountTotal
		sale.TotalAmount += sale.Items[index].LineTotal

		if err := tx.Table("products").
			Where("store_id = ? AND id = ?", sale.StoreID, product.ID).
			Updates(map[string]any{
				"quantity":   gorm.Expr("quantity - ?", item.Quantity),
				"updated_at": sale.CreatedAt,
			}).Error; err != nil {
			return Sale{}, err
		}
	}

	sale.SubtotalAmount = roundMoney(sale.SubtotalAmount)
	itemDiscountAmount := roundMoney(sale.DiscountAmount)
	payableBeforeBillDiscount := roundMoney(sale.TotalAmount)
	if sale.BillDiscountAmount < 0 {
		return Sale{}, ErrInvalidBillDiscount
	}
	if sale.BillDiscountAmount > payableBeforeBillDiscount {
		return Sale{}, ErrBillDiscountExceedsAmount
	}
	sale.BillDiscountAmount = roundMoney(sale.BillDiscountAmount)
	sale.DiscountAmount = roundMoney(itemDiscountAmount + sale.BillDiscountAmount)
	afterDiscount := roundMoney(payableBeforeBillDiscount - sale.BillDiscountAmount)
	if sale.VATPercent < 0 {
		sale.VATPercent = 0
	}
	if sale.VATIncluded {
		if sale.VATPercent > 0 {
			sale.VATAmount = roundMoney(afterDiscount * sale.VATPercent / (100 + sale.VATPercent))
		}
		sale.TotalAmount = afterDiscount
	} else {
		if sale.VATPercent > 0 {
			sale.VATAmount = roundMoney(afterDiscount * sale.VATPercent / 100)
		}
		sale.TotalAmount = roundMoney(afterDiscount + sale.VATAmount)
	}
	sale.ChangeAmount = roundMoney(sale.PaidAmount - sale.TotalAmount)
	if sale.ChangeAmount < 0 {
		return Sale{}, ErrInvalidPaidAmount
	}

	salePayload := map[string]any{
		"id":                       sale.ID,
		"store_id":                 sale.StoreID,
		"sale_number":              sale.SaleNumber,
		"cashier_user_id":          sale.CashierUserID,
		"status":                   sale.Status,
		"customer_id":              sale.CustomerID,
		"customer_level":           sale.CustomerLevel,
		"network_discount_percent": sale.NetworkDiscountPercent,
		"total_items":              sale.TotalItems,
		"subtotal_amount":          sale.SubtotalAmount,
		"discount_amount":          sale.DiscountAmount,
		"bill_discount_amount":     sale.BillDiscountAmount,
		"vat_included":             sale.VATIncluded,
		"vat_percent":              sale.VATPercent,
		"vat_amount":               sale.VATAmount,
		"total_amount":             sale.TotalAmount,
		"paid_amount":              sale.PaidAmount,
		"change_amount":            sale.ChangeAmount,
		"sold_at":                  sale.SoldAt,
		"created_at":               sale.CreatedAt,
	}
	if sale.PaymentMethod == "" {
		salePayload["payment_method"] = nil
	} else {
		salePayload["payment_method"] = sale.PaymentMethod
	}
	if sale.Note == "" {
		salePayload["note"] = nil
	} else {
		salePayload["note"] = sale.Note
	}
	if sale.CustomerID == "" {
		salePayload["customer_id"] = nil
		salePayload["customer_level"] = nil
	}
	if err := tx.Table("sales").Create(salePayload).Error; err != nil {
		return Sale{}, err
	}

	for _, item := range sale.Items {
		itemPayload := map[string]any{
			"id":                       item.ID,
			"sale_id":                  item.SaleID,
			"product_id":               item.ProductID,
			"product_name":             item.ProductName,
			"quantity":                 item.Quantity,
			"unit_price":               item.UnitPrice,
			"discount_value":           item.DiscountValue,
			"discount_amount_per_unit": item.DiscountAmountPerUnit,
			"line_subtotal":            item.LineSubtotal,
			"line_discount_total":      item.LineDiscountTotal,
			"line_total":               item.LineTotal,
			"created_at":               item.CreatedAt,
		}
		if item.SKU == "" {
			itemPayload["sku"] = nil
		} else {
			itemPayload["sku"] = item.SKU
		}
		if item.UnitType == "" {
			itemPayload["unit_type"] = nil
		} else {
			itemPayload["unit_type"] = item.UnitType
		}
		if item.DiscountType == "" {
			itemPayload["discount_type"] = nil
		} else {
			itemPayload["discount_type"] = item.DiscountType
		}
		if err := tx.Table("sale_items").Create(itemPayload).Error; err != nil {
			return Sale{}, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return Sale{}, err
	}
	return sale, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Sale, error) {
	promptPaySelect := "'' AS store_promptpay_id"
	if hasColumn, err := r.hasStorePromptPayIDColumn(ctx); err == nil && hasColumn {
		promptPaySelect = "COALESCE(st.promptpay_id, '') AS store_promptpay_id"
	}

	var sales []Sale
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(fmt.Sprintf("s.id, s.store_id, st.name AS store_name, COALESCE(st.address, '') AS store_address, COALESCE(st.phone, '') AS store_phone, %s, s.sale_number, s.cashier_user_id, COALESCE(u.full_name, '') AS cashier_name, s.status, s.payment_method, s.note, s.customer_id, COALESCE(c.full_name, '') AS customer_name, COALESCE(c.phone, '') AS customer_phone, s.customer_level, s.network_discount_percent, s.total_items, s.subtotal_amount, s.discount_amount, COALESCE(s.bill_discount_amount, 0) AS bill_discount_amount, s.vat_included, s.vat_percent, s.vat_amount, s.total_amount, s.paid_amount, s.change_amount, s.sold_at, s.created_at", promptPaySelect)).
		Joins("JOIN stores st ON st.id = s.store_id").
		Joins("LEFT JOIN users u ON u.id = s.cashier_user_id").
		Joins("LEFT JOIN customers c ON c.id = s.customer_id").
		Where("s.store_id = ?", storeID).
		Order("s.sold_at DESC, s.created_at DESC").
		Find(&sales).Error
	return sales, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, saleID string) (Sale, error) {
	promptPaySelect := "'' AS store_promptpay_id"
	if hasColumn, err := r.hasStorePromptPayIDColumn(ctx); err == nil && hasColumn {
		promptPaySelect = "COALESCE(st.promptpay_id, '') AS store_promptpay_id"
	}

	var sale Sale
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(fmt.Sprintf("s.id, s.store_id, st.name AS store_name, COALESCE(st.address, '') AS store_address, COALESCE(st.phone, '') AS store_phone, %s, s.sale_number, s.cashier_user_id, COALESCE(u.full_name, '') AS cashier_name, s.status, s.payment_method, s.note, s.customer_id, COALESCE(c.full_name, '') AS customer_name, COALESCE(c.phone, '') AS customer_phone, s.customer_level, s.network_discount_percent, s.total_items, s.subtotal_amount, s.discount_amount, COALESCE(s.bill_discount_amount, 0) AS bill_discount_amount, s.vat_included, s.vat_percent, s.vat_amount, s.total_amount, s.paid_amount, s.change_amount, s.sold_at, s.created_at", promptPaySelect)).
		Joins("JOIN stores st ON st.id = s.store_id").
		Joins("LEFT JOIN users u ON u.id = s.cashier_user_id").
		Joins("LEFT JOIN customers c ON c.id = s.customer_id").
		Where("s.store_id = ? AND s.id = ?", storeID, saleID).
		Take(&sale).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Sale{}, ErrSaleNotFound
		}
		return Sale{}, err
	}
	if err := r.db.WithContext(ctx).
		Model(&SaleItem{}).
		Where("sale_id = ?", sale.ID).
		Order("created_at ASC").
		Find(&sale.Items).Error; err != nil {
		return Sale{}, err
	}
	return sale, nil
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

func (r PostgresRepository) lockProductForSale(ctx context.Context, tx *gorm.DB, storeID, productID string) (productSnapshot, error) {
	var product productSnapshot
	err := tx.WithContext(ctx).
		Table("products p").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("p.id, p.name, COALESCE(p.sku, '') AS sku, COALESCE(pu.name, '') AS unit_type, p.quantity, p.is_active, p.base_price, p.special_price, p.special_price_start_at, p.special_price_end_at").
		Joins("LEFT JOIN product_units pu ON pu.id = p.product_unit_id").
		Where("p.store_id = ? AND p.id = ?", storeID, productID).
		Take(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return productSnapshot{}, ErrProductNotFound
		}
		return productSnapshot{}, err
	}
	return product, nil
}

func calculateDiscount(discountType string, discountValue *float64, unitPrice float64) (float64, error) {
	normalizedType := normalizeDiscountType(discountType)
	if normalizedType == "" {
		if discountValue != nil {
			return 0, ErrInvalidDiscountType
		}
		return 0, nil
	}
	if discountValue == nil {
		return 0, ErrDiscountValueRequired
	}
	if *discountValue < 0 {
		return 0, ErrInvalidDiscountValue
	}

	switch normalizedType {
	case DiscountTypeAmount:
		if *discountValue > unitPrice {
			return 0, ErrAmountDiscountExceedsPrice
		}
		return *discountValue, nil
	case DiscountTypePercent:
		if *discountValue > 100 {
			return 0, ErrInvalidPercentDiscount
		}
		return unitPrice * (*discountValue / 100), nil
	default:
		return 0, ErrInvalidDiscountType
	}
}

func calculateNetworkDiscount(percent, unitPrice, manualDiscount float64) float64 {
	if percent <= 0 {
		return 0
	}
	base := unitPrice - manualDiscount
	if base <= 0 {
		return 0
	}
	return base * (percent / 100)
}

func (r PostgresRepository) hasStorePromptPayIDColumn(ctx context.Context) (bool, error) {
	type columnLookup struct {
		Exists bool `gorm:"column:exists"`
	}

	var lookup columnLookup
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = current_schema()
					AND table_name = 'stores'
					AND column_name = 'promptpay_id'
			) AS exists
		`).
		Scan(&lookup).Error
	if err != nil {
		return false, err
	}
	return lookup.Exists, nil
}
