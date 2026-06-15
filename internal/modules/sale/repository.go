package sale

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, sale Sale, discount DiscountInput) (Sale, error)
	FindByIdempotencyKey(ctx context.Context, storeID, key string) (*Sale, error)
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

	// Phase W4B: resolve + validate + LOCK the single effective sale-point location inside the
	// tx (authoritative): the explicit request location, else the store default sale location.
	saleLocationID, err := r.resolveAndLockSaleLocation(ctx, tx, sale.StoreID, sale.LocationID, discount.IsElevated)
	if err != nil {
		return Sale{}, err
	}
	sale.LocationID = saleLocationID

	// Group cart quantities by product (duplicate lines → one deduction) and deduct ONLY from
	// the sale location, in deterministic product-id order (deadlock-safe). Stock is taken from
	// the one sale point — never aggregated across locations, never from storage, never a fallback.
	grouped := map[string]int{}
	productOrder := make([]string, 0)
	for _, item := range sale.Items {
		pid := strings.TrimSpace(item.ProductID)
		if _, ok := grouped[pid]; !ok {
			productOrder = append(productOrder, pid)
		}
		grouped[pid] += item.Quantity
	}
	sort.Strings(productOrder)
	snapshots := map[string]productSnapshot{}
	for _, pid := range productOrder {
		product, err := r.lockProductForSale(ctx, tx, sale.StoreID, pid)
		if err != nil {
			return Sale{}, err
		}
		if !product.IsActive {
			return Sale{}, ErrProductInactive
		}
		snapshots[pid] = product
		if err := r.checkAndDeductAtLocation(ctx, tx, sale.StoreID, pid, grouped[pid], sale.ID, sale.CashierUserID, saleLocationID); err != nil {
			return Sale{}, err
		}
	}

	// Pricing per request line (sale_items stay per-line) reusing the locked product snapshots.
	for index, item := range sale.Items {
		product := snapshots[strings.TrimSpace(item.ProductID)]
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
		"location_id":              sale.LocationID,
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
	if sale.IdempotencyKey == "" {
		salePayload["idempotency_key"] = nil
		salePayload["request_fingerprint"] = nil
	} else {
		salePayload["idempotency_key"] = sale.IdempotencyKey
		salePayload["request_fingerprint"] = sale.RequestFingerprint
	}
	if err := tx.Table("sales").Create(salePayload).Error; err != nil {
		// A concurrent request carrying the same Idempotency-Key won the race to the
		// unique index. Surface it as an idempotency conflict; the SERVICE then re-reads
		// the committed winner and, when the intent matches, returns that original sale
		// (this tx is rolled back by defer, so its second deduction never persists).
		if isIdempotencyKeyViolation(err) {
			return Sale{}, ErrSaleIdempotencyConflict
		}
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

// FindByIdempotencyKey returns the sale previously persisted under the given store +
// Idempotency-Key, or (nil, nil) when the key is empty or no such sale exists. The
// service uses it to short-circuit a retried sale-create before opening a transaction.
func (r PostgresRepository) FindByIdempotencyKey(ctx context.Context, storeID, key string) (*Sale, error) {
	if strings.TrimSpace(key) == "" {
		return nil, nil
	}
	var sale Sale
	err := r.db.WithContext(ctx).
		Table("sales").
		Where("store_id = ? AND idempotency_key = ?", storeID, key).
		Take(&sale).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sale, nil
}

// resolveAndLockSaleLocation determines, validates, and LOCKS the single effective
// sale-point location for this sale inside the transaction (authoritative). Resolution
// policy (Phase W4B): an explicit request location, otherwise the store's default sale
// location. The chosen location row is locked FOR UPDATE and re-checked under the lock
// (same store, active, is_sale_point, parent warehouse active) so a concurrent
// deactivation cannot slip a sale through after a stale pre-check. There is NO fallback
// to another location and NO aggregation across locations.
func (r PostgresRepository) resolveAndLockSaleLocation(ctx context.Context, tx *gorm.DB, storeID, requested string, isElevated bool) (string, error) {
	requested = strings.TrimSpace(requested)
	// The store default sale location is needed when no explicit location was given, and
	// also to authorize a non-elevated actor's explicit pick (§17: cashiers sell only at
	// the store default; owner/manager may select any active sale point).
	if requested == "" || !isElevated {
		var def struct {
			ID string `gorm:"column:id"`
		}
		err := tx.WithContext(ctx).
			Table("locations").
			Select("id").
			Where("store_id = ? AND is_default_sale = true AND is_active = true AND is_sale_point = true", storeID).
			Order("id").
			Take(&def).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", ErrNoSaleLocation
			}
			return "", err
		}
		if requested == "" {
			requested = def.ID
		} else if !isElevated && requested != def.ID {
			// A cashier may not sell from a non-default sale point.
			return "", ErrSaleLocationInvalid
		}
	}

	var loc struct {
		StoreID     string `gorm:"column:store_id"`
		IsActive    bool   `gorm:"column:is_active"`
		IsSalePoint bool   `gorm:"column:is_sale_point"`
		WarehouseID string `gorm:"column:warehouse_id"`
	}
	err := tx.WithContext(ctx).
		Table("locations").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("store_id, is_active, is_sale_point, COALESCE(warehouse_id, '') AS warehouse_id").
		Where("id = ?", requested).
		Take(&loc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrSaleLocationInvalid
		}
		return "", err
	}
	if loc.StoreID != storeID {
		return "", ErrSaleLocationCrossStore
	}
	if !loc.IsActive || !loc.IsSalePoint {
		return "", ErrSaleLocationInvalid
	}
	// The parent warehouse must also be active — a deactivated warehouse takes its
	// locations offline for selling. Lock it FOR UPDATE to close the same TOCTOU window.
	if loc.WarehouseID != "" {
		var wh struct {
			IsActive bool `gorm:"column:is_active"`
		}
		err := tx.WithContext(ctx).
			Table("warehouses").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("is_active").
			Where("id = ?", loc.WarehouseID).
			Take(&wh).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", ErrSaleLocationInvalid
			}
			return "", err
		}
		if !wh.IsActive {
			return "", ErrSaleLocationInvalid
		}
	}
	return requested, nil
}

// checkAndDeductAtLocation locks the product's stock row at the resolved sale location
// FOR UPDATE, rejects the whole sale when that single location lacks enough on hand
// (no aggregation, no fallback, no negative stock), then deducts exactly qty with a
// quantity-guarded UPDATE and records one paired SALE movement at that location.
func (r PostgresRepository) checkAndDeductAtLocation(ctx context.Context, tx *gorm.DB, storeID, productID string, qty int, saleID, cashierUserID, locationID string) error {
	var locked struct {
		Quantity int `gorm:"column:quantity"`
	}
	err := tx.WithContext(ctx).
		Table("stocks").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("COALESCE(quantity, 0) AS quantity").
		Where("product_id = ? AND location_id = ?", productID, locationID).
		Take(&locked).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No stock row at the sale point at all → nothing available to sell here.
			return InsufficientSaleStockError{ProductID: productID, Available: 0}
		}
		return err
	}
	if locked.Quantity < qty {
		return InsufficientSaleStockError{ProductID: productID, Available: locked.Quantity}
	}

	// Guarded decrement: the WHERE quantity >= qty makes a negative result impossible
	// even if the lock were somehow bypassed; RowsAffected==0 means it would go negative.
	result := tx.WithContext(ctx).Exec(`
			UPDATE stocks
			SET quantity = quantity - ?, updated_at = NOW()
			WHERE product_id = ? AND location_id = ? AND quantity >= ?
		`, qty, productID, locationID, qty)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return InsufficientSaleStockError{ProductID: productID, Available: locked.Quantity}
	}

	now := gorm.Expr("NOW()")
	if err := tx.Table("stock_movements").Create(map[string]any{
		"id":              newStockMovementID(),
		"store_id":        storeID,
		"product_id":      productID,
		"location_id":     locationID,
		"quantity_change": -qty,
		"type":            "SALE",
		"reference_id":    saleID,
		"note":            "sale deduction",
		"created_by":      cashierUserID,
		"created_at":      now,
		"updated_at":      now,
	}).Error; err != nil {
		return err
	}
	return nil
}

// isIdempotencyKeyViolation reports a unique-constraint error (SQLSTATE 23505) raised
// specifically by the per-store idempotency-key index. Scoping to the index NAME means an
// unrelated unique collision (primary key, sale_number) is NOT mis-mapped to an
// idempotency conflict — it surfaces as a real 500 instead.
func isIdempotencyKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") && strings.Contains(msg, "sales_store_idempotency_key")
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
		Select(fmt.Sprintf("s.id, s.store_id, s.location_id, st.name AS store_name, COALESCE(st.address, '') AS store_address, COALESCE(st.phone, '') AS store_phone, %s, s.sale_number, s.cashier_user_id, COALESCE(u.full_name, '') AS cashier_name, s.status, s.payment_method, s.note, s.customer_id, COALESCE(c.full_name, '') AS customer_name, COALESCE(c.phone, '') AS customer_phone, s.customer_level, s.network_discount_percent, s.total_items, s.subtotal_amount, s.discount_amount, COALESCE(s.bill_discount_amount, 0) AS bill_discount_amount, s.vat_included, s.vat_percent, s.vat_amount, s.total_amount, s.paid_amount, s.change_amount, s.sold_at, s.created_at", storeExtra)).
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
		Select(fmt.Sprintf("s.id, s.store_id, s.location_id, st.name AS store_name, COALESCE(st.address, '') AS store_address, COALESCE(st.phone, '') AS store_phone, %s, s.sale_number, s.cashier_user_id, COALESCE(u.full_name, '') AS cashier_name, s.status, s.payment_method, s.note, s.customer_id, COALESCE(c.full_name, '') AS customer_name, COALESCE(c.phone, '') AS customer_phone, s.customer_level, s.network_discount_percent, s.total_items, s.subtotal_amount, s.discount_amount, COALESCE(s.bill_discount_amount, 0) AS bill_discount_amount, s.vat_included, s.vat_percent, s.vat_amount, s.total_amount, s.paid_amount, s.change_amount, s.sold_at, s.created_at", storeExtra)).
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
	// status <> 'suspended' mirrors the member module's canonical access check
	// (member/repository.go) so a suspended store member cannot operate the POS.
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ? AND status <> 'suspended'", storeID, userID, []string{"owner", "manager", "cashier"}).
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
		Where("store_id = ? AND user_id = ? AND role IN ? AND status <> 'suspended'", storeID, userID, []string{"owner", "manager"}).
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
