package creditsale

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pos-backend/internal/idgen"
	"pos-backend/internal/modules/sale"
)

type Repository interface {
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
	CreateReceivable(ctx context.Context, cs CreditSale, initial *CreditPayment) (CreditSale, error)
	List(ctx context.Context, storeID string) ([]CreditSale, error)
	Get(ctx context.Context, storeID, creditSaleID string) (CreditSale, error)
	AddPayment(ctx context.Context, storeID string, p CreditPayment) (CreditSale, error)
	Cancel(ctx context.Context, storeID, creditSaleID, actorUserID string) (CreditSale, error)
	Summary(ctx context.Context, storeID string) (DebtSummary, error)
	StatementContext(ctx context.Context, storeID, creditSaleID string) (StatementContext, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ? AND status <> 'suspended'", storeID, userID, []string{"owner", "manager", "cashier"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ── Read ────────────────────────────────────────────────────────────────────

const creditSaleSelect = "cs.*, COALESCE(c.full_name, '') AS customer_name, COALESCE(c.phone, '') AS customer_phone"

func (r PostgresRepository) baseQuery(ctx context.Context, storeID string) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("credit_sales cs").
		Select(creditSaleSelect).
		Joins("LEFT JOIN customers c ON c.id = cs.customer_id").
		Where("cs.store_id = ?", storeID)
}

func (r PostgresRepository) List(ctx context.Context, storeID string) ([]CreditSale, error) {
	var sales []CreditSale
	if err := r.baseQuery(ctx, storeID).Order("cs.created_at DESC").Find(&sales).Error; err != nil {
		return nil, err
	}
	if err := r.hydrate(ctx, sales); err != nil {
		return nil, err
	}
	for i := range sales {
		sales[i].Status = displayStatus(sales[i])
	}
	return sales, nil
}

func (r PostgresRepository) Get(ctx context.Context, storeID, creditSaleID string) (CreditSale, error) {
	var cs CreditSale
	err := r.baseQuery(ctx, storeID).Where("cs.id = ?", creditSaleID).Take(&cs).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreditSale{}, ErrNotFound
		}
		return CreditSale{}, err
	}
	one := []CreditSale{cs}
	if err := r.hydrate(ctx, one); err != nil {
		return CreditSale{}, err
	}
	one[0].Status = displayStatus(one[0])
	return one[0], nil
}

// hydrate loads each receivable's items (from the underlying sale's sale_items)
// and its payment timeline (credit_payments) in two batched queries.
func (r PostgresRepository) hydrate(ctx context.Context, sales []CreditSale) error {
	if len(sales) == 0 {
		return nil
	}
	saleIDs := make([]string, 0, len(sales))
	creditIDs := make([]string, 0, len(sales))
	for _, s := range sales {
		saleIDs = append(saleIDs, s.SaleID)
		creditIDs = append(creditIDs, s.ID)
	}

	var items []CreditSaleItem
	err := r.db.WithContext(ctx).
		Table("sale_items").
		Select("sale_id, product_id, product_name, COALESCE(unit_type, '') AS unit, unit_price AS price, quantity, line_total AS total").
		Where("sale_id IN ?", saleIDs).
		Order("created_at ASC").
		Find(&items).Error
	if err != nil {
		return err
	}
	itemsBySale := make(map[string][]CreditSaleItem)
	for _, it := range items {
		itemsBySale[it.SaleID] = append(itemsBySale[it.SaleID], it)
	}

	var payments []CreditPayment
	err = r.db.WithContext(ctx).
		Table("credit_payments").
		Select("id, credit_sale_id, amount, method, note, paid_at").
		Where("credit_sale_id IN ?", creditIDs).
		Order("paid_at ASC").
		Find(&payments).Error
	if err != nil {
		return err
	}
	paymentsByCredit := make(map[string][]CreditPayment)
	for _, p := range payments {
		paymentsByCredit[p.CreditSaleID] = append(paymentsByCredit[p.CreditSaleID], p)
	}

	for i := range sales {
		if v := itemsBySale[sales[i].SaleID]; v != nil {
			sales[i].Items = v
		} else {
			sales[i].Items = []CreditSaleItem{}
		}
		if v := paymentsByCredit[sales[i].ID]; v != nil {
			sales[i].Payments = v
		} else {
			sales[i].Payments = []CreditPayment{}
		}
	}
	return nil
}

// ── Create ──────────────────────────────────────────────────────────────────

func (r PostgresRepository) CreateReceivable(ctx context.Context, cs CreditSale, initial *CreditPayment) (CreditSale, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return CreditSale{}, tx.Error
	}
	defer tx.Rollback()

	// Server-assigned human document number: CR + YYYYMMDD + 4-digit per-store seq.
	prefix := "CR" + time.Now().Format("20060102")
	var sameDay int64
	if err := tx.Table("credit_sales").
		Where("store_id = ? AND document_number LIKE ?", cs.StoreID, prefix+"%").
		Count(&sameDay).Error; err != nil {
		return CreditSale{}, err
	}
	cs.DocumentNumber = fmt.Sprintf("%s%04d", prefix, sameDay+1)

	if err := tx.Table("credit_sales").Create(map[string]any{
		"id":               cs.ID,
		"store_id":         cs.StoreID,
		"sale_id":          cs.SaleID,
		"customer_id":      cs.CustomerID,
		"document_number":  cs.DocumentNumber,
		"type":             cs.Type,
		"total_amount":     cs.TotalAmount,
		"paid_amount":      cs.PaidAmount,
		"remaining_amount": cs.RemainingAmount,
		"status":           cs.Status,
		"due_date":         cs.DueDate,
		"note":             cs.Note,
		"created_by":       cs.CreatedBy,
		"created_at":       cs.CreatedAt,
		"updated_at":       cs.UpdatedAt,
	}).Error; err != nil {
		return CreditSale{}, err
	}

	if initial != nil {
		if err := tx.Table("credit_payments").Create(paymentRow(*initial)).Error; err != nil {
			return CreditSale{}, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return CreditSale{}, err
	}
	return r.Get(ctx, cs.StoreID, cs.ID)
}

// ── Payment ─────────────────────────────────────────────────────────────────

func (r PostgresRepository) AddPayment(ctx context.Context, storeID string, p CreditPayment) (CreditSale, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return CreditSale{}, tx.Error
	}
	defer tx.Rollback()

	// Phase W5 — lock the receivable header FOR UPDATE so AddPayment serializes against
	// Cancel (and other payments) on the same row: a payment can never be recorded after
	// a cancellation commits, and concurrent payments can't both read a stale balance.
	var header CreditSale
	if err := tx.Table("credit_sales").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("store_id = ? AND id = ?", storeID, p.CreditSaleID).
		Take(&header).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreditSale{}, ErrNotFound
		}
		return CreditSale{}, err
	}
	if header.Status == StatusCancelled {
		return CreditSale{}, ErrPaymentAfterCancel
	}

	newPaid := roundMoney(header.PaidAmount + p.Amount)
	if newPaid > header.TotalAmount+0.001 {
		return CreditSale{}, ErrOverpayment
	}
	remaining := roundMoney(header.TotalAmount - newPaid)
	if remaining < 0 {
		remaining = 0
	}
	status := StatusPartial
	if remaining <= 0 {
		status = StatusCompleted
	}

	if err := tx.Table("credit_payments").Create(paymentRow(p)).Error; err != nil {
		return CreditSale{}, err
	}
	if err := tx.Table("credit_sales").
		Where("store_id = ? AND id = ?", storeID, p.CreditSaleID).
		Updates(map[string]any{
			"paid_amount":      newPaid,
			"remaining_amount": remaining,
			"status":           status,
			"updated_at":       time.Now().UTC().Format(time.RFC3339),
		}).Error; err != nil {
		return CreditSale{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return CreditSale{}, err
	}
	return r.Get(ctx, storeID, p.CreditSaleID)
}

// ── Cancel: restock + void underlying sale ──────────────────────────────────

func (r PostgresRepository) Cancel(ctx context.Context, storeID, creditSaleID, actorUserID string) (CreditSale, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return CreditSale{}, tx.Error
	}
	defer tx.Rollback()

	// Lock the receivable row FOR UPDATE before the status check so two concurrent
	// cancellations serialize: the first flips status to cancelled and restocks; the
	// second blocks on the lock, then re-reads 'cancelled' under the lock and is rejected
	// — stock can never be restored twice (idempotent cancellation via locked transition).
	var header CreditSale
	if err := tx.Table("credit_sales").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("store_id = ? AND id = ?", storeID, creditSaleID).
		Take(&header).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreditSale{}, ErrNotFound
		}
		return CreditSale{}, err
	}
	if header.Status == StatusCancelled {
		return CreditSale{}, ErrAlreadyCancelled
	}
	// Phase W5 — a receivable that has already collected money cannot be cancelled: the
	// system has NO payment-reversal/refund workflow, so cancelling would silently strand
	// recorded payments on a voided sale (a financial inconsistency). Block it explicitly.
	// Checked under the FOR UPDATE lock so it is consistent with concurrent AddPayment.
	if roundMoney(header.PaidAmount) > 0 {
		return CreditSale{}, ErrCannotCancelPaid
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	// 1. Restock by reversing the exact SALE stock movements of the underlying sale.
	type mv struct {
		ProductID      string `gorm:"column:product_id"`
		LocationID     string `gorm:"column:location_id"`
		QuantityChange int    `gorm:"column:quantity_change"`
	}
	var movements []mv
	if err := tx.Table("stock_movements").
		Select("product_id, location_id, quantity_change").
		Where("reference_id = ? AND type = ?", header.SaleID, "SALE").
		// Deterministic lock order (product_id, location_id) so the restock loop acquires
		// `stocks` row locks in the SAME order as sale.Create's sorted product-id loop —
		// closes the cross-cancel and cancel-vs-sale deadlock window (no behavior change).
		Order("product_id, location_id").
		Find(&movements).Error; err != nil {
		return CreditSale{}, err
	}
	for _, m := range movements {
		restore := -m.QuantityChange // SALE rows are negative; restore the positive amount
		if restore <= 0 {
			continue
		}
		// Restock the originating location. If the stocks row was removed after the
		// sale (e.g. via a warehouse product removal), UPDATE matches 0 rows — do
		// NOT silently drop the returned goods. Recreate the row via UPSERT. If the
		// location itself no longer exists the INSERT fails the FK and the whole
		// cancel transaction rolls back with an explicit error (no false RETURN).
		res := tx.Exec(`UPDATE stocks SET quantity = quantity + ?, updated_at = NOW() WHERE product_id = ? AND location_id = ?`,
			restore, m.ProductID, m.LocationID)
		if res.Error != nil {
			return CreditSale{}, res.Error
		}
		if res.RowsAffected == 0 {
			if err := tx.Exec(`INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, NOW(), NOW())
				ON CONFLICT (product_id, location_id) DO UPDATE SET quantity = stocks.quantity + EXCLUDED.quantity, updated_at = NOW()`,
				idgen.Generate(idgen.PrefixStock), storeID, m.ProductID, m.LocationID, restore).Error; err != nil {
				return CreditSale{}, err
			}
		}
		if err := tx.Table("stock_movements").Create(map[string]any{
			"id":              idgen.Generate(idgen.PrefixStockMovement),
			"store_id":        storeID,
			"product_id":      m.ProductID,
			"location_id":     m.LocationID,
			"quantity_change": restore,
			"type":            "RETURN",
			"reference_id":    creditSaleID,
			"note":            "credit sale cancellation restock",
			"created_by":      actorUserID,
			"created_at":      now,
			"updated_at":      now,
		}).Error; err != nil {
			return CreditSale{}, err
		}
	}

	// 2. Void the underlying sale so its revenue + COGS drop out of all finance
	//    reports. The goods were returned to stock in step 1, so recognising no
	//    revenue/COGS is the correct accounting — no bad-debt expense is booked.
	//    sale_items are left intact so the cancelled receivable still shows what
	//    was sold, and the customer's collected payments remain on record.
	if err := tx.Table("sales").
		Where("id = ? AND store_id = ?", header.SaleID, storeID).
		Update("status", sale.SaleStatusVoided).Error; err != nil {
		return CreditSale{}, err
	}

	// 3. Mark the receivable cancelled.
	if err := tx.Table("credit_sales").
		Where("store_id = ? AND id = ?", storeID, creditSaleID).
		Updates(map[string]any{
			"status":       StatusCancelled,
			"cancelled_at": nowStr,
			"updated_at":   nowStr,
		}).Error; err != nil {
		return CreditSale{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return CreditSale{}, err
	}
	return r.Get(ctx, storeID, creditSaleID)
}

// ── Summary ─────────────────────────────────────────────────────────────────

func (r PostgresRepository) Summary(ctx context.Context, storeID string) (DebtSummary, error) {
	var s DebtSummary
	today := time.Now().Format("2006-01-02")
	err := r.db.WithContext(ctx).
		Table("credit_sales").
		Select(`
			COALESCE(SUM(remaining_amount) FILTER (WHERE status <> 'cancelled' AND remaining_amount > 0), 0) AS total_outstanding,
			COUNT(*) FILTER (WHERE status <> 'cancelled' AND remaining_amount > 0) AS open_count,
			COUNT(*) FILTER (WHERE status NOT IN ('cancelled','completed') AND remaining_amount > 0 AND due_date <> '' AND due_date < ?) AS overdue_count,
			COALESCE(SUM(remaining_amount) FILTER (WHERE status NOT IN ('cancelled','completed') AND remaining_amount > 0 AND due_date <> '' AND due_date < ?), 0) AS overdue_amount
		`, today, today).
		Where("store_id = ?", storeID).
		Take(&s).Error
	return s, err
}

// ── Statement ───────────────────────────────────────────────────────────────

// StatementContext gathers the seller (store) + customer header and ALL of that
// customer's non-cancelled credit sales (with payment timelines) for the PDF.
func (r PostgresRepository) StatementContext(ctx context.Context, storeID, creditSaleID string) (StatementContext, error) {
	trigger, err := r.Get(ctx, storeID, creditSaleID)
	if err != nil {
		return StatementContext{}, err
	}

	var store struct {
		Name    string `gorm:"column:name"`
		Address string `gorm:"column:address"`
		TaxID   string `gorm:"column:tax_id"`
	}
	if err := r.db.WithContext(ctx).
		Raw("SELECT name, COALESCE(address, '') AS address, COALESCE(tax_id, '') AS tax_id FROM stores WHERE id = ?", storeID).
		Scan(&store).Error; err != nil {
		return StatementContext{}, err
	}

	var cust struct {
		Address string `gorm:"column:address"`
	}
	if err := r.db.WithContext(ctx).
		Raw("SELECT COALESCE(address, '') AS address FROM customers WHERE id = ? AND store_id = ?", trigger.CustomerID, storeID).
		Scan(&cust).Error; err != nil {
		return StatementContext{}, err
	}

	var sales []CreditSale
	if err := r.baseQuery(ctx, storeID).
		Where("cs.customer_id = ? AND cs.status <> ?", trigger.CustomerID, StatusCancelled).
		Order("cs.created_at ASC").
		Find(&sales).Error; err != nil {
		return StatementContext{}, err
	}
	if err := r.hydrate(ctx, sales); err != nil {
		return StatementContext{}, err
	}
	for i := range sales {
		sales[i].Status = displayStatus(sales[i])
	}

	return StatementContext{
		StoreName:    store.Name,
		StoreAddress: store.Address,
		StoreTaxID:   store.TaxID,
		CustomerName: trigger.CustomerName,
		CustomerAddr: cust.Address,
		DocumentNo:   trigger.DocumentNumber,
		Sales:        sales,
	}, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func paymentRow(p CreditPayment) map[string]any {
	return map[string]any{
		"id":             p.ID,
		"credit_sale_id": p.CreditSaleID,
		"store_id":       p.StoreID,
		"amount":         p.Amount,
		"method":         p.Method,
		"note":           p.Note,
		"paid_at":        p.PaidAt,
		"created_by":     p.CreatedBy,
	}
}

func roundMoney(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// displayStatus overlays the time-derived "overdue" state on the stored status,
// matching the frontend computeStatus precedence (cancelled > completed > overdue
// > partial > pending).
func displayStatus(cs CreditSale) string {
	if cs.Status == StatusCancelled || cs.Status == StatusCompleted {
		return cs.Status
	}
	if cs.RemainingAmount > 0 && cs.DueDate != "" && cs.DueDate < time.Now().Format("2006-01-02") {
		return StatusOverdue
	}
	return cs.Status
}
