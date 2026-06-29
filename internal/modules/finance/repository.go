package finance

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// expenseApprovedStatus mirrors expense.StatusApproved — only approved, un-voided
// expenses count as operating cost in the P&L (kept local to avoid a module dep).
const expenseApprovedStatus = "approved"

// saleVoidedStatus mirrors sale.SaleStatusVoided — voided sales (e.g. a cancelled
// credit sale whose goods were restocked) are excluded from all revenue/COGS
// reporting. Kept local to avoid a module dependency (same rationale as above).
const saleVoidedStatus = "voided"

// notLoanSaleSQL excludes sales backing a loan (ยืมสินค้า) receivable. Borrowed goods
// are not revenue: lending creates a sale row only to deduct stock, and returning the
// goods (ReturnGoods) restocks them WITHOUT voiding that sale — so loans would otherwise
// inflate revenue/COGS/payment-mix. Real credit sales (credit_sales.type='credit') stay
// counted. Correlate on the sales alias `s`; every aggregate below aliases sales as s.
const notLoanSaleSQL = `NOT EXISTS (SELECT 1 FROM credit_sales cs WHERE cs.sale_id = s.id AND cs.type = 'loan')`

type Repository interface {
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
	GetRevenue(ctx context.Context, storeID string, from, to time.Time) (Revenue, error)
	GetCOGS(ctx context.Context, storeID string, from, to time.Time) (COGS, error)
	GetOperatingExpenses(ctx context.Context, storeID string, from, to time.Time) (float64, error)
	GetExpenseByCategory(ctx context.Context, storeID string, from, to time.Time) ([]CategoryTotal, error)
	GetPaymentBreakdown(ctx context.Context, storeID string, from, to time.Time) ([]PaymentMethodStat, error)

	GetInventorySnapshot(ctx context.Context, storeID string) (InventorySnapshot, error)
	GetDeadStock(ctx context.Context, storeID string, soldBefore time.Time) ([]DeadStockItem, int64, float64, error)
	GetStockVelocity(ctx context.Context, storeID string) ([]StockVelocityItem, error)
	GetOverstockItems(ctx context.Context, storeID string) ([]OverstockItem, error)
	GetTopProducts(ctx context.Context, storeID string, from, to time.Time, limit int) ([]TopProduct, error)
	GetSalesTrend(ctx context.Context, storeID string, from, to time.Time) ([]TrendPoint, int64, error)
	GetSalesByMonth(ctx context.Context, storeID string, from, to time.Time) ([]MonthlyStat, error)
	GetSalesCounters(ctx context.Context, storeID string, from, to time.Time) (int64, int64, error)
	GetCategoryBreakdown(ctx context.Context, storeID string, from, to time.Time) ([]CategoryTotal, error)
	GetSalesByHour(ctx context.Context, storeID string, from, to time.Time) ([]HourStat, error)
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

// GetRevenue sums sale totals over [from, to), excluding voided sales.
// Refunds from sale_returns during the same window are summed separately so the
// caller can compute net revenue = gross - refunds.
func (r PostgresRepository) GetRevenue(ctx context.Context, storeID string, from, to time.Time) (Revenue, error) {
	var result Revenue
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(`
			COUNT(*) AS sales_count,
			COALESCE(SUM(s.total_amount), 0) AS gross_revenue,
			COALESCE(SUM(s.discount_amount), 0) AS discount_amount,
			COALESCE(SUM(s.vat_amount), 0) AS vat_amount
		`).
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Scan(&result).Error
	if err != nil {
		return result, err
	}

	var refunds float64
	if err := r.db.WithContext(ctx).
		Table("sale_returns sr").
		Select("COALESCE(SUM(sr.refund_amount), 0)").
		Where("sr.store_id = ? AND sr.created_at >= ? AND sr.created_at < ?", storeID, from, to).
		Scan(&refunds).Error; err != nil {
		return result, err
	}
	result.Refunds = refunds
	return result, nil
}

// GetCOGS values every sold line at its snapshotted sale-time cost
// (sale_items.unit_cost), falling back to the product's current cost for historical
// rows. A line whose product was deleted (product_id NULL) or has no resolvable
// cost counts toward MissingCostLines and contributes 0 to the total.
func (r PostgresRepository) GetCOGS(ctx context.Context, storeID string, from, to time.Time) (COGS, error) {
	var result COGS
	err := r.db.WithContext(ctx).
		Table("sale_items si").
		Select(`
			COALESCE(SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price, 0)), 0) AS total,
			COALESCE(SUM(CASE WHEN COALESCE(si.unit_cost, p.cost_price, 0) <= 0 THEN 1 ELSE 0 END), 0) AS missing_cost_lines
		`).
		Joins("JOIN sales s ON s.id = si.sale_id").
		Joins("LEFT JOIN products p ON p.id = si.product_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Scan(&result).Error
	return result, err
}

// ceilDate rounds a timestamp UP to the next midnight (or returns it unchanged
// if already at midnight) — used as the exclusive upper date bound for expenses.
func ceilDate(t time.Time) time.Time {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if t.Equal(day) {
		return day
	}
	return day.Add(24 * time.Hour)
}

// expenseWindow scopes approved, un-voided expenses by expense_date (a DATE
// column). The upper bound is the day AFTER `to` so that today's expenses are
// included for "ending now" period ranges (where to = now, mid-day), matching
// how the sales side counts today's sales. Custom ranges (to already at next
// midnight) stay correct: the to-day is included.
func (r PostgresRepository) expenseWindow(ctx context.Context, storeID string, from, to time.Time) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("expenses").
		Where("expenses.store_id = ? AND expenses.voided_at IS NULL AND expenses.status = ?", storeID, expenseApprovedStatus).
		Where("expenses.expense_date >= ? AND expenses.expense_date < ?", from.Format("2006-01-02"), ceilDate(to).Format("2006-01-02"))
}

func (r PostgresRepository) GetOperatingExpenses(ctx context.Context, storeID string, from, to time.Time) (float64, error) {
	var total float64
	err := r.expenseWindow(ctx, storeID, from, to).
		Select("COALESCE(SUM(amount), 0) AS total").
		Scan(&total).Error
	return total, err
}

// GetExpenseByCategory returns expense totals grouped by category.
// NOTE: category names may appear garbled (mojibake) if the postgres_data volume was created without
// UTF-8 encoding. The amounts and IDs are unaffected. Fix requires a pg_dump backup + volume recreate.
// See AGENTS.md "Known Issues / DATABASE ENCODING" for the full procedure.
func (r PostgresRepository) GetExpenseByCategory(ctx context.Context, storeID string, from, to time.Time) ([]CategoryTotal, error) {
	var rows []CategoryTotal
	err := r.expenseWindow(ctx, storeID, from, to).
		Select("expenses.category_id AS category_id, COALESCE(ec.name, '') AS name, COALESCE(SUM(expenses.amount), 0) AS total").
		Joins("LEFT JOIN expense_categories ec ON ec.id = expenses.category_id").
		Group("expenses.category_id, ec.name").
		Order("total DESC").
		Find(&rows).Error
	return rows, err
}

func (r PostgresRepository) GetPaymentBreakdown(ctx context.Context, storeID string, from, to time.Time) ([]PaymentMethodStat, error) {
	var items []PaymentMethodStat
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(`
			`+canonicalPaymentMethodSQL+` AS payment_method,
			COUNT(*) AS sales_count,
			COALESCE(SUM(s.total_amount), 0) AS amount
		`).
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group(canonicalPaymentMethodSQL).
		Order("amount DESC").
		Find(&items).Error
	return items, err
}

// canonicalPaymentMethodSQL collapses legacy / duplicate payment_method values onto the
// canonical channel keys (mirrors lib/payment-method.ts + dashboard repo):
//   transfer → bank_transfer · qr → promptpay · credit_card|debit_card → card.
// `credit` (sold-on-credit) is kept distinct — it is not a tender.
const canonicalPaymentMethodSQL = `CASE
	WHEN NULLIF(TRIM(s.payment_method), '') IS NULL THEN 'unknown'
	WHEN LOWER(TRIM(s.payment_method)) = 'transfer' THEN 'bank_transfer'
	WHEN LOWER(TRIM(s.payment_method)) = 'qr' THEN 'promptpay'
	WHEN LOWER(TRIM(s.payment_method)) IN ('credit_card','debit_card') THEN 'card'
	ELSE LOWER(TRIM(s.payment_method))
END`

// ── Executive summary aggregates ──────────────────────────────────────────────

// GetInventorySnapshot is a single aggregate scan over active products (current
// snapshot, no period). Status thresholds mirror the Inventory page:
// out = stock<=0, low = 0<stock<=min_stock, inStock = stock>min_stock.
// InventoryValue is the cost-basis tied capital (cost_price*stock) over in-stock.
func (r PostgresRepository) GetInventorySnapshot(ctx context.Context, storeID string) (InventorySnapshot, error) {
	var snap InventorySnapshot
	err := r.db.WithContext(ctx).
		Table("product_view").
		Select(`
			COALESCE(SUM(CASE WHEN total_stock > 0 THEN cost_price * total_stock ELSE 0 END), 0) AS inventory_value,
			COUNT(*) FILTER (WHERE total_stock > min_stock)                              AS in_stock,
			COUNT(*) FILTER (WHERE total_stock > 0 AND total_stock <= min_stock)         AS low_stock,
			COUNT(*) FILTER (WHERE total_stock <= 0)                                     AS out_of_stock,
			COUNT(*) FILTER (WHERE total_stock > 0 AND COALESCE(cost_price, 0) <= 0)     AS missing_cost
		`).
		Where("store_id = ? AND is_active = TRUE AND deleted_at IS NULL", storeID).
		Scan(&snap).Error
	return snap, err
}

// GetDeadStock returns the in-stock active products whose last sale predates
// soldBefore (or that have never sold), ordered by tied capital at cost. The count
// and total tied value are derived from the same list so they always agree. One
// LEFT JOIN against a per-product last-sold aggregate — no N+1.
func (r PostgresRepository) GetDeadStock(ctx context.Context, storeID string, soldBefore time.Time) ([]DeadStockItem, int64, float64, error) {
	lastSold := r.db.WithContext(ctx).
		Table("sale_items si").
		Select("si.product_id AS product_id, MAX(s.sold_at) AS last_sold").
		Joins("JOIN sales s ON s.id = si.sale_id").
		Where("s.store_id = ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, saleVoidedStatus).
		Group("si.product_id")

	inner := r.db.WithContext(ctx).
		Table("product_view pv").
		Select(`
			pv.id AS product_id,
			pv.name AS product_name,
			pv.total_stock AS remaining,
			COALESCE(pv.cost_price, 0) * pv.total_stock AS tied_value,
			ls.last_sold AS last_sold,
			(ls.last_sold IS NULL OR ls.last_sold < ?) AS dead
		`, soldBefore).
		Joins("LEFT JOIN (?) ls ON ls.product_id = pv.id", lastSold).
		Where("pv.store_id = ? AND pv.is_active = TRUE AND pv.total_stock > 0 AND pv.deleted_at IS NULL", storeID)

	var items []DeadStockItem
	err := r.db.WithContext(ctx).
		Table("(?) AS t", inner).
		Select("product_id, product_name, remaining, tied_value, last_sold").
		Where("dead").
		Order("tied_value DESC, product_name ASC").
		Find(&items).Error
	if err != nil {
		return nil, 0, 0, err
	}

	var count int64
	var value float64
	for i := range items {
		count++
		value += items[i].TiedValue
		items[i].NeverSold = items[i].LastSold == nil
	}
	return items, count, value, nil
}

// GetTopProducts ranks products by units sold and includes per-product profit
// (line revenue − quantity × COALESCE(unit_cost, current product cost)). Lines
// whose product was deleted (product_id NULL) contribute 0 cost.
func (r PostgresRepository) GetTopProducts(ctx context.Context, storeID string, from, to time.Time, limit int) ([]TopProduct, error) {
	var items []TopProduct
	err := r.db.WithContext(ctx).
		Table("sale_items si").
		Select(`
			si.product_id,
			si.product_name,
			COALESCE(SUM(si.quantity), 0) AS quantity_sold,
			COALESCE(SUM(si.line_total), 0) AS revenue,
			COALESCE(SUM(si.line_total), 0) - COALESCE(SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price, 0)), 0) AS profit
		`).
		Joins("JOIN sales s ON s.id = si.sale_id").
		Joins("LEFT JOIN products p ON p.id = si.product_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group("si.product_id, si.product_name").
		Order("quantity_sold DESC, revenue DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

// GetSalesCounters returns total units sold (SUM sale_items.quantity) and the
// number of distinct identified customers (walk-in/NULL customers are ignored).
func (r PostgresRepository) GetSalesCounters(ctx context.Context, storeID string, from, to time.Time) (int64, int64, error) {
	var units int64
	err := r.db.WithContext(ctx).
		Table("sale_items si").
		Joins("JOIN sales s ON s.id = si.sale_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Select("COALESCE(SUM(si.quantity), 0)").
		Scan(&units).Error
	if err != nil {
		return 0, 0, err
	}

	var customers int64
	err = r.db.WithContext(ctx).
		Table("sales s").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Select("COUNT(DISTINCT s.customer_id)").
		Scan(&customers).Error
	return units, customers, err
}

// GetCategoryBreakdown sums line revenue per product category over the window.
// Uncategorised / deleted-product lines fall under an empty name (labelled
// client-side).
func (r PostgresRepository) GetCategoryBreakdown(ctx context.Context, storeID string, from, to time.Time) ([]CategoryTotal, error) {
	var rows []CategoryTotal
	err := r.db.WithContext(ctx).
		Table("sale_items si").
		Select("COALESCE(p.product_type_id, '') AS category_id, COALESCE(pt.name, '') AS name, COALESCE(SUM(si.line_total), 0) AS total").
		Joins("JOIN sales s ON s.id = si.sale_id").
		Joins("LEFT JOIN products p ON p.id = si.product_id").
		Joins("LEFT JOIN product_types pt ON pt.id = p.product_type_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group("COALESCE(p.product_type_id, ''), pt.name").
		Order("total DESC").
		Find(&rows).Error
	return rows, err
}

// GetSalesByHour buckets revenue and order count by hour-of-day (0–23) in the
// store's Asia/Bangkok timezone, so a 12:00 local sale lands in bucket 12 (not 05
// as raw UTC would). Sparse: only hours with sales are returned; the client fills
// the 24-hour grid.
func (r PostgresRepository) GetSalesByHour(ctx context.Context, storeID string, from, to time.Time) ([]HourStat, error) {
	var rows []HourStat
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select("EXTRACT(HOUR FROM s.sold_at AT TIME ZONE 'Asia/Bangkok')::int AS hour, COALESCE(SUM(s.total_amount), 0) AS revenue, COUNT(*) AS orders").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group("EXTRACT(HOUR FROM s.sold_at AT TIME ZONE 'Asia/Bangkok')").
		Order("hour ASC").
		Find(&rows).Error
	return rows, err
}

// GetSalesTrend returns sparse per-day {revenue, cogs, profit} buckets plus the
// total order count. Revenue/orders come from sales (one row per sale); COGS from
// sale_items; merged by day in Go. Gap days are filled client-side from the range.
func (r PostgresRepository) GetSalesTrend(ctx context.Context, storeID string, from, to time.Time) ([]TrendPoint, int64, error) {
	type revRow struct {
		Day     string  `gorm:"column:day"`
		Revenue float64 `gorm:"column:revenue"`
		Orders  int64   `gorm:"column:orders"`
	}
	var revRows []revRow
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM-DD') AS day, COALESCE(SUM(s.total_amount), 0) AS revenue, COUNT(*) AS orders").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM-DD')").
		Order("day ASC").
		Find(&revRows).Error
	if err != nil {
		return nil, 0, err
	}

	type cogsRow struct {
		Day  string  `gorm:"column:day"`
		COGS float64 `gorm:"column:cogs"`
	}
	var cogsRows []cogsRow
	err = r.db.WithContext(ctx).
		Table("sale_items si").
		Select("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM-DD') AS day, COALESCE(SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price, 0)), 0) AS cogs").
		Joins("JOIN sales s ON s.id = si.sale_id").
		Joins("LEFT JOIN products p ON p.id = si.product_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM-DD')").
		Find(&cogsRows).Error
	if err != nil {
		return nil, 0, err
	}

	cogsByDay := make(map[string]float64, len(cogsRows))
	for _, c := range cogsRows {
		cogsByDay[c.Day] = c.COGS
	}

	// Deduct per-day refunds so trend revenue matches P&L net revenue methodology.
	type refundRow struct {
		Day     string  `gorm:"column:day"`
		Refunds float64 `gorm:"column:refunds"`
	}
	var refundRows []refundRow
	err = r.db.WithContext(ctx).
		Table("sale_returns sr").
		Select("to_char(sr.created_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM-DD') AS day, COALESCE(SUM(sr.refund_amount), 0) AS refunds").
		Where("sr.store_id = ? AND sr.created_at >= ? AND sr.created_at < ?", storeID, from, to).
		Group("to_char(sr.created_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM-DD')").
		Find(&refundRows).Error
	if err != nil {
		return nil, 0, err
	}
	refundsByDay := make(map[string]float64, len(refundRows))
	for _, rf := range refundRows {
		refundsByDay[rf.Day] = rf.Refunds
	}

	points := make([]TrendPoint, 0, len(revRows))
	var totalOrders int64
	for _, rv := range revRows {
		cogs := cogsByDay[rv.Day]
		netRev := rv.Revenue - refundsByDay[rv.Day]
		points = append(points, TrendPoint{
			Day:     rv.Day,
			Revenue: netRev,
			COGS:    cogs,
			Profit:  netRev - cogs,
		})
		totalOrders += rv.Orders
	}
	return points, totalOrders, nil
}

// GetSalesByMonth aggregates the same revenue/COGS/refund/discount data as GetSalesTrend
// but bucketed per calendar month (Asia/Bangkok) for the per-month P&L summary table.
// Revenue is net of refunds; Profit is gross profit (revenue − COGS).
func (r PostgresRepository) GetSalesByMonth(ctx context.Context, storeID string, from, to time.Time) ([]MonthlyStat, error) {
	type saleRow struct {
		Month    string  `gorm:"column:month"`
		Revenue  float64 `gorm:"column:revenue"`
		Orders   int64   `gorm:"column:orders"`
		Discount float64 `gorm:"column:discount"`
	}
	var saleRows []saleRow
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM') AS month, COALESCE(SUM(s.total_amount), 0) AS revenue, COUNT(*) AS orders, COALESCE(SUM(s.discount_amount), 0) AS discount").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM')").
		Order("month ASC").
		Find(&saleRows).Error
	if err != nil {
		return nil, err
	}

	type cogsRow struct {
		Month string  `gorm:"column:month"`
		COGS  float64 `gorm:"column:cogs"`
	}
	var cogsRows []cogsRow
	err = r.db.WithContext(ctx).
		Table("sale_items si").
		Select("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM') AS month, COALESCE(SUM(si.quantity * COALESCE(si.unit_cost, p.cost_price, 0)), 0) AS cogs").
		Joins("JOIN sales s ON s.id = si.sale_id").
		Joins("LEFT JOIN products p ON p.id = si.product_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ? AND "+notLoanSaleSQL, storeID, from, to, saleVoidedStatus).
		Group("to_char(s.sold_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM')").
		Find(&cogsRows).Error
	if err != nil {
		return nil, err
	}
	cogsByMonth := make(map[string]float64, len(cogsRows))
	for _, c := range cogsRows {
		cogsByMonth[c.Month] = c.COGS
	}

	type refundRow struct {
		Month   string  `gorm:"column:month"`
		Refunds float64 `gorm:"column:refunds"`
	}
	var refundRows []refundRow
	err = r.db.WithContext(ctx).
		Table("sale_returns sr").
		Select("to_char(sr.created_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM') AS month, COALESCE(SUM(sr.refund_amount), 0) AS refunds").
		Where("sr.store_id = ? AND sr.created_at >= ? AND sr.created_at < ?", storeID, from, to).
		Group("to_char(sr.created_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM')").
		Find(&refundRows).Error
	if err != nil {
		return nil, err
	}
	refundsByMonth := make(map[string]float64, len(refundRows))
	for _, rf := range refundRows {
		refundsByMonth[rf.Month] = rf.Refunds
	}

	months := make([]MonthlyStat, 0, len(saleRows))
	for _, sr := range saleRows {
		cogs := cogsByMonth[sr.Month]
		netRev := sr.Revenue - refundsByMonth[sr.Month]
		months = append(months, MonthlyStat{
			Month:    sr.Month,
			Orders:   sr.Orders,
			Revenue:  netRev,
			COGS:     cogs,
			Profit:   netRev - cogs,
			Discount: sr.Discount,
		})
	}
	return months, nil
}

// GetStockVelocity returns the top 20 active in-stock products ordered by urgency
// (fewest days of stock first). avg_daily_sales is computed over the last 30 days;
// days_of_stock is NULL when there are no recent sales (no velocity data).
func (r PostgresRepository) GetStockVelocity(ctx context.Context, storeID string) ([]StockVelocityItem, error) {
	var items []StockVelocityItem
	err := r.db.WithContext(ctx).Raw(`
		WITH velocity AS (
			SELECT si.product_id,
			       COALESCE(SUM(si.quantity), 0)::float / 30.0 AS avg_daily_sales
			FROM sale_items si
			JOIN sales s ON s.id = si.sale_id
			WHERE s.store_id = ?
			  AND s.sold_at >= NOW() - INTERVAL '30 days'
			  AND s.status <> 'voided'
			  AND NOT EXISTS (SELECT 1 FROM credit_sales cs WHERE cs.sale_id = s.id AND cs.type = 'loan')
			GROUP BY si.product_id
		)
		SELECT
		    pv.id                                                      AS product_id,
		    pv.name                                                    AS product_name,
		    COALESCE(pv.total_stock, 0)::bigint                       AS current_stock,
		    COALESCE(v.avg_daily_sales, 0)                            AS avg_daily_sales,
		    CASE WHEN COALESCE(v.avg_daily_sales, 0) > 0
		         THEN COALESCE(pv.total_stock, 0)::float / v.avg_daily_sales
		         ELSE NULL END                                         AS days_of_stock,
		    COALESCE(pv.min_stock, 0)::bigint                         AS min_stock,
		    COALESCE(pv.max_stock, 0)::bigint                         AS max_stock,
		    COALESCE(pv.cost_price, 0)                                AS cost_price
		FROM product_view pv
		LEFT JOIN velocity v ON v.product_id = pv.id
		WHERE pv.store_id = ?
		  AND pv.is_active = TRUE
		  AND pv.deleted_at IS NULL
		  AND COALESCE(pv.total_stock, 0) > 0
		ORDER BY
		    CASE WHEN COALESCE(v.avg_daily_sales, 0) > 0
		         THEN COALESCE(pv.total_stock, 0)::float / v.avg_daily_sales
		         ELSE 9999999 END ASC,
		    pv.name ASC
		LIMIT 20
	`, storeID, storeID).Scan(&items).Error
	return items, err
}

// GetOverstockItems returns active products whose current stock exceeds max_stock.
// Only products with max_stock > 0 are included (unset threshold = no cap).
// Ordered by capital value (excess units × cost) descending.
func (r PostgresRepository) GetOverstockItems(ctx context.Context, storeID string) ([]OverstockItem, error) {
	var items []OverstockItem
	err := r.db.WithContext(ctx).
		Table("product_view pv").
		Select(`
			pv.id                                                      AS product_id,
			pv.name                                                    AS product_name,
			COALESCE(pv.total_stock, 0)::bigint                       AS current_stock,
			pv.max_stock::bigint                                       AS max_stock,
			(COALESCE(pv.total_stock, 0) - pv.max_stock)::bigint      AS overstock_qty,
			COALESCE(pv.cost_price, 0)                                AS cost_price,
			(COALESCE(pv.total_stock, 0) - pv.max_stock) * COALESCE(pv.cost_price, 0) AS capital_value
		`).
		Where(
			"pv.store_id = ? AND pv.is_active = TRUE AND pv.deleted_at IS NULL AND COALESCE(pv.max_stock, 0) > 0 AND COALESCE(pv.total_stock, 0) > pv.max_stock",
			storeID,
		).
		Order("capital_value DESC, pv.name ASC").
		Limit(10).
		Find(&items).Error
	return items, err
}
