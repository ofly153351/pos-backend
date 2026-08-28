package invoice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/platform/taxcalc"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, invoice Invoice) (Invoice, error)
	ListByStore(ctx context.Context, storeID string) ([]Invoice, error)
	GetByID(ctx context.Context, storeID, invoiceID string) (Invoice, error)
	AddPayment(ctx context.Context, storeID, invoiceID string, payment InvoicePayment) (Invoice, error)
	MarkUnpaid(ctx context.Context, storeID, invoiceID, actorUserID, reason string, atTime time.Time) (Invoice, error)
	GetPaymentProof(ctx context.Context, storeID, invoiceID, paymentID string) (InvoicePayment, error)
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

		invoice.Items[index].ID = newInvoiceItemID()
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
		invoice.Items[index].LineSubtotal = roundMoney(invoice.Items[index].LineSubtotal)
		invoice.Items[index].LineDiscountTotal = roundMoney(invoice.Items[index].LineDiscountTotal)
		invoice.Items[index].LineTotal = roundMoney(invoice.Items[index].LineTotal)
		invoice.Items[index].CreatedAt = invoice.CreatedAt
		invoice.SubtotalAmount += invoice.Items[index].LineSubtotal
		invoice.DiscountAmount += invoice.Items[index].LineDiscountTotal
		invoice.TotalAmount += invoice.Items[index].LineTotal

		// Find a sale point location to deduct from
		var deductLocID string
		if err := tx.Table("locations").
			Where("store_id = ? AND is_sale_point = ?", invoice.StoreID, true).
			Order("created_at ASC").
			Select("id").
			Take(&deductLocID).Error; err != nil {
			return Invoice{}, err
		}

		// Deduct from stocks table
		stockResult := tx.Exec(
			`UPDATE stocks SET quantity = quantity - ?, updated_at = NOW()
			 WHERE store_id = ? AND product_id = ? AND location_id = ? AND quantity >= ?`,
			item.Quantity, invoice.StoreID, product.ID, deductLocID, item.Quantity,
		)
		if stockResult.Error != nil {
			return Invoice{}, stockResult.Error
		}
		if stockResult.RowsAffected == 0 {
			return Invoice{}, fmt.Errorf("insufficient stock for product %s", item.ProductID)
		}

		// Create stock movement record
		if err := tx.Table("stock_movements").Create(map[string]any{
			"id":              newStockMovementID(),
			"store_id":        invoice.StoreID,
			"product_id":      product.ID,
			"location_id":     deductLocID,
			"type":            "SALE",
			"quantity_change": -item.Quantity,
			"reference_id":    invoice.ID,
			"note":            "invoice deduction",
			"created_by":      invoice.CashierUserID,
			"created_at":      invoice.CreatedAt,
			"updated_at":      invoice.CreatedAt,
		}).Error; err != nil {
			return Invoice{}, err
		}
	}

	invoice.SubtotalAmount = roundMoney(invoice.SubtotalAmount)
	invoice.DiscountAmount = roundMoney(invoice.DiscountAmount)
	afterDiscount := roundMoney(invoice.TotalAmount)
	if invoice.VATPercent < 0 {
		invoice.VATPercent = 0
	}
	// Canonical VAT formula, shared with the sale module and the /vat/calculate preview.
	invoice.VATAmount, invoice.TotalAmount = taxcalc.ComputeVAT(afterDiscount, invoice.VATPercent, invoice.VATIncluded)
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
		"vat_included":             invoice.VATIncluded,
		"vat_percent":              invoice.VATPercent,
		"vat_amount":               invoice.VATAmount,
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
		Select("i.id, i.store_id, st.name AS store_name, i.invoice_number, i.customer_id, COALESCE(c.full_name, '') AS customer_name, i.cashier_user_id, i.status, i.payment_method, i.note, i.due_at, i.customer_level, i.network_discount_percent, i.total_items, i.subtotal_amount, i.discount_amount, COALESCE(i.vat_included, TRUE) AS vat_included, COALESCE(i.vat_percent, 7) AS vat_percent, COALESCE(i.vat_amount, 0) AS vat_amount, i.total_amount, i.paid_amount, i.remaining_amount, i.created_at, i.updated_at").
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
		Select("i.id, i.store_id, st.name AS store_name, i.invoice_number, i.customer_id, COALESCE(c.full_name, '') AS customer_name, i.cashier_user_id, i.status, i.payment_method, i.note, i.due_at, i.customer_level, i.network_discount_percent, i.total_items, i.subtotal_amount, i.discount_amount, COALESCE(i.vat_included, TRUE) AS vat_included, COALESCE(i.vat_percent, 7) AS vat_percent, COALESCE(i.vat_amount, 0) AS vat_amount, i.total_amount, i.paid_amount, i.remaining_amount, i.created_at, i.updated_at").
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
		"id":                payment.ID,
		"invoice_id":        inv.ID,
		"paid_amount":       payment.PaidAmount,
		"payment_method":    payment.PaymentMethod,
		"note":              payment.Note,
		"proof_url":         nil,
		"proof_mime_type":   nil,
		"proof_file_name":   nil,
		"is_voided":         false,
		"voided_at":         nil,
		"voided_by_user_id": nil,
		"void_reason":       nil,
		"paid_at":           payment.PaidAt,
		"created_at":        payment.CreatedAt,
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

func (r PostgresRepository) MarkUnpaid(ctx context.Context, storeID, invoiceID, actorUserID, reason string, atTime time.Time) (Invoice, error) {
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
	if inv.Status == StatusUnpaid && inv.PaidAmount == 0 {
		return Invoice{}, ErrInvoiceAlreadyUnpaid
	}

	if err := tx.Table("invoice_payments").
		Where("invoice_id = ? AND is_voided = FALSE", inv.ID).
		Updates(map[string]any{
			"is_voided":         true,
			"voided_at":         atTime,
			"voided_by_user_id": actorUserID,
			"void_reason":       reason,
		}).Error; err != nil {
		return Invoice{}, err
	}

	if err := tx.Table("invoices").
		Where("id = ?", inv.ID).
		Updates(map[string]any{
			"status":           StatusUnpaid,
			"paid_amount":      0,
			"remaining_amount": inv.TotalAmount,
			"payment_method":   nil,
			"updated_at":       atTime,
		}).Error; err != nil {
		return Invoice{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Invoice{}, err
	}
	return r.GetByID(ctx, storeID, invoiceID)
}

func (r PostgresRepository) GetPaymentProof(ctx context.Context, storeID, invoiceID, paymentID string) (InvoicePayment, error) {
	var payment InvoicePayment
	err := r.db.WithContext(ctx).
		Table("invoice_payments ip").
		Select("ip.id, ip.invoice_id, ip.paid_amount, ip.payment_method, ip.note, ip.proof_url, ip.proof_mime_type, ip.proof_file_name, ip.is_voided, ip.voided_at, ip.voided_by_user_id, ip.void_reason, ip.paid_at, ip.created_at").
		Joins("JOIN invoices i ON i.id = ip.invoice_id").
		Where("i.store_id = ? AND i.id = ? AND ip.id = ?", storeID, invoiceID, paymentID).
		Take(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InvoicePayment{}, ErrInvoiceNotFound
		}
		return InvoicePayment{}, err
	}
	return payment, nil
}

func (r PostgresRepository) lockProductForInvoice(ctx context.Context, tx *gorm.DB, storeID, productID string) (productSnapshot, error) {
	var product productSnapshot
	err := tx.WithContext(ctx).
		Table("products p").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("p.id, p.name, COALESCE(p.sku, '') AS sku, COALESCE((SELECT pu.name FROM product_units pu WHERE pu.id = p.product_unit_id), '') AS unit_type, COALESCE((SELECT SUM(s.quantity) FROM stocks s WHERE s.product_id = p.id), 0) AS quantity, p.is_active, p.base_price, p.special_price, p.special_price_start_at, p.special_price_end_at").
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
