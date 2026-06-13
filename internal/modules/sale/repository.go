package sale

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, sale Sale, discount DiscountInput) (Sale, error)
	ListByStore(ctx context.Context, storeID string) ([]Sale, error)
	GetByID(ctx context.Context, storeID, saleID string) (Sale, error)
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, sale Sale, discount DiscountInput) (Sale, error) {
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

		// Check stock availability from sale-point locations
		if err := r.checkAndDeductSaleStock(ctx, tx, sale.StoreID, item.ProductID, item.Quantity, sale.ID, sale.CashierUserID); err != nil {
			return Sale{}, err
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

		sale.Items[index].ID = newSaleItemID()
		sale.Items[index].SaleID = sale.ID
		sale.Items[index].ProductName = product.Name
		sale.Items[index].SKU = product.SKU
		sale.Items[index].UnitType = product.UnitType
		sale.Items[index].UnitPrice = unitPrice
		// Snapshot the cost at sale time so historical COGS does not drift when the
		// product's cost_price later changes (migration 025 added sale_items.unit_cost).
		sale.Items[index].UnitCost = product.CostPrice
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
	}

	sale.SubtotalAmount = roundMoney(sale.SubtotalAmount)
	itemDiscountAmount := roundMoney(sale.DiscountAmount)
	payableBeforeBillDiscount := roundMoney(sale.TotalAmount)
	billDiscount, appliedPromo, err := r.resolveBillDiscount(ctx, tx, sale.StoreID, payableBeforeBillDiscount, sale.TotalItems, discount)
	if err != nil {
		return Sale{}, err
	}
	sale.BillDiscountAmount = billDiscount
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
		// Credit sales defer payment: the unpaid balance is tracked as a receivable
		// by the creditsale module, so an under-payment (including paid_amount=0) is
		// valid ONLY for payment_method='credit'. Cash/transfer still require full
		// payment. Change is clamped to 0 — nothing is owed back to the customer.
		if sale.PaymentMethod != "credit" {
			return Sale{}, ErrInvalidPaidAmount
		}
		sale.ChangeAmount = 0
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
			"unit_cost":                item.UnitCost,
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

	// Record promotion usage (ledger + aggregate counters) atomically with the sale.
	if err := r.recordPromotionUsage(ctx, tx, sale.StoreID, sale.ID, appliedPromo); err != nil {
		return Sale{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Sale{}, err
	}
	return sale, nil
}

// checkAndDeductSaleStock checks stock availability in sale-point locations
// and deducts proportionally from them, creating stock_movement records.
func (r PostgresRepository) checkAndDeductSaleStock(ctx context.Context, tx *gorm.DB, storeID, productID string, qty int, saleID, cashierUserID string) error {
	// Lock and check total available stock in sale-point locations
	var totalAvailable int
	// Lock individual rows first (FOR UPDATE can't be used with aggregate functions)
	type lockedStock struct {
		Quantity int
	}
	var lockedRows []lockedStock
	err := tx.WithContext(ctx).
		Table("stocks").
		Select("stocks.quantity").
		Joins("JOIN locations ON locations.id = stocks.location_id").
		Where("stocks.product_id = ? AND locations.store_id = ? AND locations.is_sale_point = true AND stocks.quantity > 0", productID, storeID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&lockedRows).Error
	if err != nil {
		return err
	}
	for _, row := range lockedRows {
		totalAvailable += row.Quantity
	}
	if totalAvailable < qty {
		return fmt.Errorf("%w for product %s", ErrInsufficientStock, productID)
	}

	// Get all sale-point locations with stock for this product, ordered
	type locationStock struct {
		LocationID string
		Quantity   int
	}
	var locationStocks []locationStock
	err = tx.WithContext(ctx).
		Table("stocks").
		Select("stocks.location_id, stocks.quantity").
		Joins("JOIN locations ON locations.id = stocks.location_id").
		Where("stocks.product_id = ? AND locations.store_id = ? AND locations.is_sale_point = true AND stocks.quantity > 0", productID, storeID).
		Order("stocks.quantity DESC").
		Find(&locationStocks).Error
	if err != nil {
		return err
	}

	remaining := qty
	movementID := newStockMovementID()
	now := gorm.Expr("NOW()")

	for _, ls := range locationStocks {
		if remaining <= 0 {
			break
		}
		deduct := ls.Quantity
		if deduct > remaining {
			deduct = remaining
		}

		// Update stock quantity
		result := tx.WithContext(ctx).
			Exec(`
				UPDATE stocks
				SET quantity = quantity - ?, updated_at = NOW()
				WHERE product_id = ? AND location_id = ? AND quantity >= ?
			`, deduct, productID, ls.LocationID, deduct)
		if result.Error != nil {
			return result.Error
		}

		// Create stock movement record
		if err := tx.Table("stock_movements").Create(map[string]any{
			"id":              newStockMovementID(),
			"store_id":        storeID,
			"product_id":      productID,
			"location_id":     ls.LocationID,
			"quantity_change": -deduct,
			"type":            "SALE",
			"reference_id":    saleID,
			"note":            "sale deduction",
			"created_by":      cashierUserID,
			"created_at":      now,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}

		remaining -= deduct
		_ = movementID
	}

	if remaining > 0 {
		return fmt.Errorf("%w for product %s", ErrInsufficientStock, productID)
	}

	return nil
}

func (r PostgresRepository) buildStoreExtraSelect(ctx context.Context) string {
	promptPay := "'' AS store_promptpay_id"
	if hasColumn, err := r.hasStorePromptPayIDColumn(ctx); err == nil && hasColumn {
		promptPay = "COALESCE(st.promptpay_id, '') AS store_promptpay_id"
	}
	taxID := "'' AS store_tax_id"
	if hasColumn, err := r.hasStoreTaxIDColumn(ctx); err == nil && hasColumn {
		taxID = "COALESCE(st.tax_id, '') AS store_tax_id"
	}
	// logo_url is part of the base stores schema — always present.
	logoURL := "COALESCE(st.logo_url, '') AS store_logo_url"
	return promptPay + ", " + taxID + ", " + logoURL
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Sale, error) {
	storeExtra := r.buildStoreExtraSelect(ctx)

	var sales []Sale
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(fmt.Sprintf("s.id, s.store_id, st.name AS store_name, COALESCE(st.address, '') AS store_address, COALESCE(st.phone, '') AS store_phone, %s, s.sale_number, s.cashier_user_id, COALESCE(u.full_name, '') AS cashier_name, s.status, s.payment_method, s.note, s.customer_id, COALESCE(c.full_name, '') AS customer_name, COALESCE(c.phone, '') AS customer_phone, s.customer_level, s.network_discount_percent, s.total_items, s.subtotal_amount, s.discount_amount, COALESCE(s.bill_discount_amount, 0) AS bill_discount_amount, s.vat_included, s.vat_percent, s.vat_amount, s.total_amount, s.paid_amount, s.change_amount, s.sold_at, s.created_at", storeExtra)).
		Joins("JOIN stores st ON st.id = s.store_id").
		Joins("LEFT JOIN users u ON u.id = s.cashier_user_id").
		Joins("LEFT JOIN customers c ON c.id = s.customer_id").
		Where("s.store_id = ?", storeID).
		Order("s.sold_at DESC, s.created_at DESC").
		Find(&sales).Error
	return sales, err
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, saleID string) (Sale, error) {
	storeExtra := r.buildStoreExtraSelect(ctx)

	var sale Sale
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(fmt.Sprintf("s.id, s.store_id, st.name AS store_name, COALESCE(st.address, '') AS store_address, COALESCE(st.phone, '') AS store_phone, %s, s.sale_number, s.cashier_user_id, COALESCE(u.full_name, '') AS cashier_name, s.status, s.payment_method, s.note, s.customer_id, COALESCE(c.full_name, '') AS customer_name, COALESCE(c.phone, '') AS customer_phone, s.customer_level, s.network_discount_percent, s.total_items, s.subtotal_amount, s.discount_amount, COALESCE(s.bill_discount_amount, 0) AS bill_discount_amount, s.vat_included, s.vat_percent, s.vat_amount, s.total_amount, s.paid_amount, s.change_amount, s.sold_at, s.created_at", storeExtra)).
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

// UserCanManageStore reports whether the user is owner/manager of the store (or a
// platform admin) — used to lift the cashier manual-discount cap.
func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager"}).
		Count(&count).Error
	return count > 0, err
}

// resolveBillDiscount turns the request's (manual, promo, promotion_ids) — or the
// deprecated discount_bill fallback — into a single validated bill discount.
// The promo portion is verified against active, in-window promotions for the store
// and capped at their theoretical max; the manual portion is capped (20% of subtotal
// for cashiers, remaining subtotal for owner/manager) and an over-cap manual is
// REJECTED with an explicit error (never silently reduced). Final never exceeds subtotal.
func (r PostgresRepository) resolveBillDiscount(ctx context.Context, tx *gorm.DB, storeID string, subtotal float64, totalQty int, in DiscountInput) (float64, appliedPromoResult, error) {
	var manualReq, promoReq float64
	var promoIDs []string
	if in.ManualDiscount == nil && in.PromoDiscount == nil && len(in.PromotionIDs) == 0 {
		// Legacy client: discount_bill is treated as a manual discount (same cap).
		manualReq = in.LegacyBill
	} else {
		if in.ManualDiscount != nil {
			manualReq = *in.ManualDiscount
		}
		if in.PromoDiscount != nil {
			promoReq = *in.PromoDiscount
		}
		promoIDs = in.PromotionIDs
	}
	// Reject non-finite or negative inputs explicitly. NaN comparisons are always
	// false, so without this guard a NaN would slip past the cap check and corrupt
	// the persisted total (and every downstream finance aggregate).
	if badMoney(manualReq) || badMoney(promoReq) || manualReq < 0 || promoReq < 0 {
		return 0, appliedPromoResult{}, ErrInvalidBillDiscount
	}

	// Verify the promo discount: honored only up to the theoretical max of the
	// referenced active, in-window, non-deleted promotions. No promotion ids =>
	// promo rejected. validPromoIDs are the ones that actually backed the discount.
	verifiedPromo := 0.0
	var validPromoIDs []string
	if promoReq > 0 && len(promoIDs) > 0 {
		maxPromo, valid, err := r.sumPromoCeilings(ctx, tx, storeID, promoIDs, subtotal, totalQty)
		if err != nil {
			return 0, appliedPromoResult{}, err
		}
		verifiedPromo = promoReq
		if maxPromo < verifiedPromo {
			verifiedPromo = maxPromo
		}
		validPromoIDs = valid
	}
	verifiedPromo = roundMoney(verifiedPromo)
	if verifiedPromo > subtotal {
		verifiedPromo = subtotal
	}

	// Manual cap: remaining subtotal for owner/manager; 20% for cashier. `subtotal`
	// here is payableBeforeBillDiscount (cart total after item-level discounts), so
	// the bill discount stacks on top of any item discounts.
	manualCap := roundMoney(subtotal - verifiedPromo)
	if !in.IsElevated {
		cashierCap := roundMoney(subtotal * 0.20)
		if cashierCap < manualCap {
			manualCap = cashierCap
		}
	}
	// Both sides rounded to 2dp: an over-cap manual is an EXPLICIT error, never a
	// silent reduction.
	if roundMoney(manualReq) > manualCap {
		return 0, appliedPromoResult{}, ErrManualDiscountExceedsCap
	}

	final := roundMoney(verifiedPromo + manualReq)
	if final > subtotal {
		final = subtotal
	}
	applied := appliedPromoResult{}
	if verifiedPromo > 0 && len(validPromoIDs) > 0 {
		applied = appliedPromoResult{VerifiedDiscount: verifiedPromo, PromotionIDs: validPromoIDs}
	}
	return final, applied, nil
}

// badMoney reports a non-finite float (NaN or ±Inf) that must never enter the money math.
func badMoney(v float64) bool { return math.IsNaN(v) || math.IsInf(v, 0) }

// sumPromoCeilings loads the referenced promotions (store-scoped, active, not
// soft-deleted), drops any outside its [startDate,endDate] window, and returns the
// sum of their theoretical-max discounts plus the ids that qualified.
func (r PostgresRepository) sumPromoCeilings(ctx context.Context, tx *gorm.DB, storeID string, promoIDs []string, subtotal float64, totalQty int) (float64, []string, error) {
	type promoRow struct {
		ID   string `gorm:"column:id"`
		Type string `gorm:"column:type"`
		Data string `gorm:"column:data"`
	}
	var rows []promoRow
	if err := tx.WithContext(ctx).
		Table("promotions").
		Select("id, type, data").
		Where("store_id = ? AND id IN ? AND status = ? AND deleted_at IS NULL", storeID, promoIDs, "active").
		Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	now := time.Now().UTC()
	total := 0.0
	valid := make([]string, 0, len(rows))
	for _, p := range rows {
		if !promoWithinWindow(p.Data, now) {
			continue
		}
		total += promoCeiling(p.Type, p.Data, subtotal, totalQty)
		valid = append(valid, p.ID)
	}
	return total, valid, nil
}

// recordPromotionUsage writes the per-promotion usage ledger and bumps the
// aggregate counters inside the sale transaction. The verified promo discount is
// split evenly across the applied promotions (exact for the common single-promo
// case; a documented approximation when several apply at once).
func (r PostgresRepository) recordPromotionUsage(ctx context.Context, tx *gorm.DB, storeID, saleID string, applied appliedPromoResult) error {
	n := len(applied.PromotionIDs)
	if n == 0 || applied.VerifiedDiscount <= 0 {
		return nil
	}
	share := roundMoney(applied.VerifiedDiscount / float64(n))
	now := time.Now().UTC()
	for _, pid := range applied.PromotionIDs {
		if err := tx.WithContext(ctx).Table("promotion_usages").Create(map[string]any{
			"id":              newPromotionUsageID(),
			"store_id":        storeID,
			"promotion_id":    pid,
			"sale_id":         saleID,
			"discount_amount": share,
			"created_at":      now,
		}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE promotions SET usage_count = usage_count + 1, discount_given_total = discount_given_total + ? WHERE id = ? AND store_id = ?`,
			share, pid, storeID,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r PostgresRepository) lockProductForSale(ctx context.Context, tx *gorm.DB, storeID, productID string) (productSnapshot, error) {
	var product productSnapshot
	err := tx.WithContext(ctx).
		Table("products p").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("p.id, p.name, COALESCE(p.sku, '') AS sku, COALESCE((SELECT pu.name FROM product_units pu WHERE pu.id = p.product_unit_id), '') AS unit_type, p.is_active, p.base_price, COALESCE(p.cost_price, 0) AS cost_price, p.special_price, p.special_price_start_at, p.special_price_end_at").
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
	return r.hasStoreColumn(ctx, "promptpay_id")
}

func (r PostgresRepository) hasStoreTaxIDColumn(ctx context.Context) (bool, error) {
	return r.hasStoreColumn(ctx, "tax_id")
}

func (r PostgresRepository) hasStoreColumn(ctx context.Context, column string) (bool, error) {
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
					AND column_name = ?
			) AS exists
		`, column).
		Scan(&lookup).Error
	if err != nil {
		return false, err
	}
	return lookup.Exists, nil
}
