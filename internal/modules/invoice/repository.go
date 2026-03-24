package invoice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, invoice Invoice) (Invoice, error)
	ListByStore(ctx context.Context, storeID string) ([]Invoice, error)
	GetByID(ctx context.Context, storeID, invoiceID string) (Invoice, error)
	AddPayment(ctx context.Context, storeID, invoiceID string, payment InvoicePayment) (Invoice, error)
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, invoice Invoice) (Invoice, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return Invoice{}, tx.Error
	}
	defer tx.Rollback()

	for index, item := range invoice.Items {
		product, err := r.lockProductForInvoice(ctx, tx, invoice.StoreID, item.ProductID)
		if err != nil {
			return Invoice{}, err
		}
		if !product.IsActive {
			return Invoice{}, ErrProductInactive
		}
		if product.Quantity < item.Quantity {
			return Invoice{}, fmt.Errorf("%w for product %s", ErrInsufficientStock, item.ProductID)
		}

		unitPrice := resolveEffectivePrice(product, invoice.CreatedAt)
		manualDiscountPerUnit, err := calculateDiscount(item.DiscountType, item.DiscountValue, unitPrice)
		if err != nil {
			return Invoice{}, err
		}
		networkDiscountPerUnit := calculateNetworkDiscount(invoice.NetworkDiscountPercent, unitPrice, manualDiscountPerUnit)
		discountAmountPerUnit := manualDiscountPerUnit + networkDiscountPerUnit
		if discountAmountPerUnit > unitPrice {
			discountAmountPerUnit = unitPrice
		}

		invoice.Items[index].ID = newID()
		invoice.Items[index].InvoiceID = invoice.ID
		invoice.Items[index].ProductName = product.Name
		invoice.Items[index].SKU = product.SKU
		invoice.Items[index].UnitType = product.UnitType
		invoice.Items[index].UnitPrice = unitPrice
		invoice.Items[index].DiscountType = normalizeDiscountType(item.DiscountType)
		invoice.Items[index].DiscountAmountPerUnit = discountAmountPerUnit
		invoice.Items[index].LineSubtotal = unitPrice * float64(item.Quantity)
		invoice.Items[index].LineDiscountTotal = discountAmountPerUnit * float64(item.Quantity)
		invoice.Items[index].LineTotal = invoice.Items[index].LineSubtotal - invoice.Items[index].LineDiscountTotal
		invoice.Items[index].CreatedAt = invoice.CreatedAt
		invoice.SubtotalAmount += invoice.Items[index].LineSubtotal
		invoice.DiscountAmount += invoice.Items[index].LineDiscountTotal
		invoice.TotalAmount += invoice.Items[index].LineTotal

		if err := tx.Table("products").
			Where("store_id = ? AND id = ?", invoice.StoreID, product.ID).
			Updates(map[string]any{
				"quantity":   gorm.Expr("quantity - ?", item.Quantity),
				"updated_at": invoice.CreatedAt,
			}).Error; err != nil {
			return Invoice{}, err
		}
	}

	invoice.RemainingAmount = invoice.TotalAmount
	invoice.PaidAmount = 0
	invoice.Status = StatusUnpaid

	payload := map[string]any{
		"id":                       invoice.ID,
		"store_id":                 invoice.StoreID,
		"invoice_number":           invoice.InvoiceNumber,
		"customer_id":              invoice.CustomerID,
		"cashier_user_id":          invoice.CashierUserID,
		"status":                   invoice.Status,
		"payment_method":           nil,
		"note":                     nil,
		"due_at":                   invoice.DueAt,
		"customer_level":           invoice.CustomerLevel,
		"network_discount_percent": invoice.NetworkDiscountPercent,
		"total_items":              invoice.TotalItems,
		"subtotal_amount":          invoice.SubtotalAmount,
		"discount_amount":          invoice.DiscountAmount,
		"total_amount":             invoice.TotalAmount,
		"paid_amount":              invoice.PaidAmount,
		"remaining_amount":         invoice.RemainingAmount,
		"created_at":               invoice.CreatedAt,
		"updated_at":               invoice.CreatedAt,
	}
	if strings.TrimSpace(invoice.Note) != "" {
		payload["note"] = invoice.Note
	}
	if strings.TrimSpace(invoice.PaymentMethod) != "" {
		payload["payment_method"] = invoice.PaymentMethod
	}
	if err := tx.Table("invoices").Create(payload).Error; err != nil {
		return Invoice{}, err
	}

	for _, item := range invoice.Items {
		itemPayload := map[string]any{
			"id":                       item.ID,
			"invoice_id":               item.InvoiceID,
			"product_id":               item.ProductID,
			"product_name":             item.ProductName,
			"quantity":                 item.Quantity,
			"unit_price":               item.UnitPrice,
			"discount_type":            nil,
			"discount_value":           item.DiscountValue,
			"discount_amount_per_unit": item.DiscountAmountPerUnit,
			"line_subtotal":            item.LineSubtotal,
			"line_discount_total":      item.LineDiscountTotal,
			"line_total":               item.LineTotal,
			"created_at":               item.CreatedAt,
			"sku":                      nil,
			"unit_type":                nil,
		}
		if item.SKU != "" {
			itemPayload["sku"] = item.SKU
		}
		if item.UnitType != "" {
			itemPayload["unit_type"] = item.UnitType
		}
		if item.DiscountType != "" {
			itemPayload["discount_type"] = item.DiscountType
		}
		if err := tx.Table("invoice_items").Create(itemPayload).Error; err != nil {
			return Invoice{}, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return Invoice{}, err
	}
	return r.GetByID(ctx, invoice.StoreID, invoice.ID)
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Invoice, error) {
	var invoices []Invoice
	err := r.db.WithContext(ctx).
		Table("invoices i").
		Select("i.id, i.store_id, st.name AS store_name, i.invoice_number, i.customer_id, COALESCE(c.full_name, '') AS customer_name, i.cashier_user_id, i.status, i.payment_method, i.note, i.due_at, i.customer_level, i.network_discount_percent, i.total_items, i.subtotal_amount, i.discount_amount, i.total_amount, i.paid_amount, i.remaining_amount, i.created_at, i.updated_at").
		Joins("JOIN stores st ON st.id = i.store_id").
		Joins("LEFT JOIN customers c ON c.id = i.customer_id").
		Where("i.store_id = ?", storeID).
		Order("i.created_at DESC").
		Find(&invoices).Error
	return invoices, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, invoiceID string) (Invoice, error) {
	var invoice Invoice
	err := r.db.WithContext(ctx).
		Table("invoices i").
		Select("i.id, i.store_id, st.name AS store_name, i.invoice_number, i.customer_id, COALESCE(c.full_name, '') AS customer_name, i.cashier_user_id, i.status, i.payment_method, i.note, i.due_at, i.customer_level, i.network_discount_percent, i.total_items, i.subtotal_amount, i.discount_amount, i.total_amount, i.paid_amount, i.remaining_amount, i.created_at, i.updated_at").
		Joins("JOIN stores st ON st.id = i.store_id").
		Joins("LEFT JOIN customers c ON c.id = i.customer_id").
		Where("i.store_id = ? AND i.id = ?", storeID, invoiceID).
		Take(&invoice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Invoice{}, ErrInvoiceNotFound
		}
		return Invoice{}, err
	}

	if err := r.db.WithContext(ctx).Model(&InvoiceItem{}).
		Where("invoice_id = ?", invoice.ID).
		Order("created_at ASC").
		Find(&invoice.Items).Error; err != nil {
		return Invoice{}, err
	}
	if err := r.db.WithContext(ctx).Model(&InvoicePayment{}).
		Where("invoice_id = ?", invoice.ID).
		Order("created_at ASC").
		Find(&invoice.Payments).Error; err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

func (r PostgresRepository) AddPayment(ctx context.Context, storeID, invoiceID string, payment InvoicePayment) (Invoice, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return Invoice{}, tx.Error
	}
	defer tx.Rollback()

	type lockedInvoice struct {
		ID              string  `gorm:"column:id"`
		StoreID         string  `gorm:"column:store_id"`
		Status          string  `gorm:"column:status"`
		TotalAmount     float64 `gorm:"column:total_amount"`
		PaidAmount      float64 `gorm:"column:paid_amount"`
		RemainingAmount float64 `gorm:"column:remaining_amount"`
	}
	var inv lockedInvoice
	err := tx.Table("invoices").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("store_id = ? AND id = ?", storeID, invoiceID).
		Take(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Invoice{}, ErrInvoiceNotFound
		}
		return Invoice{}, err
	}
	if inv.Status == StatusPaid {
		return Invoice{}, ErrInvoiceAlreadyPaid
	}
	if payment.PaidAmount > inv.RemainingAmount {
		return Invoice{}, ErrPaymentExceedsRemaining
	}

	newPaid := inv.PaidAmount + payment.PaidAmount
	newRemaining := inv.TotalAmount - newPaid
	newStatus := StatusPartiallyPaid
	if newRemaining <= 0 {
		newRemaining = 0
		newStatus = StatusPaid
	}

	if err := tx.Table("invoices").
		Where("id = ?", inv.ID).
		Updates(map[string]any{
			"paid_amount":      newPaid,
			"remaining_amount": newRemaining,
			"status":           newStatus,
			"payment_method":   payment.PaymentMethod,
			"updated_at":       payment.CreatedAt,
		}).Error; err != nil {
		return Invoice{}, err
	}

	paymentPayload := map[string]any{
		"id":              payment.ID,
		"invoice_id":      inv.ID,
		"paid_amount":     payment.PaidAmount,
		"payment_method":  payment.PaymentMethod,
		"note":            payment.Note,
		"proof_url":       nil,
		"proof_mime_type": nil,
		"proof_file_name": nil,
		"paid_at":         payment.PaidAt,
		"created_at":      payment.CreatedAt,
	}
	if strings.TrimSpace(payment.ProofURL) != "" {
		paymentPayload["proof_url"] = payment.ProofURL
	}
	if strings.TrimSpace(payment.ProofMimeType) != "" {
		paymentPayload["proof_mime_type"] = payment.ProofMimeType
	}
	if strings.TrimSpace(payment.ProofFileName) != "" {
		paymentPayload["proof_file_name"] = payment.ProofFileName
	}
	if err := tx.Table("invoice_payments").Create(paymentPayload).Error; err != nil {
		return Invoice{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Invoice{}, err
	}
	return r.GetByID(ctx, storeID, invoiceID)
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

func (r PostgresRepository) lockProductForInvoice(ctx context.Context, tx *gorm.DB, storeID, productID string) (productSnapshot, error) {
	var product productSnapshot
	err := tx.WithContext(ctx).
		Table("products").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id, name, COALESCE(sku, '') AS sku, COALESCE(unit_type, '') AS unit_type, quantity, is_active, base_price, special_price, special_price_start_at, special_price_end_at").
		Where("store_id = ? AND id = ?", storeID, productID).
		Take(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return productSnapshot{}, ErrProductNotFound
		}
		return productSnapshot{}, err
	}
	return product, nil
}
