package finance

import "time"

// PnLQuery is the parsed P&L request. Either a named Period (7d/30d/90d) or an
// explicit From/To window is used to scope every figure.
type PnLQuery struct {
	Period string
	From   *time.Time
	To     *time.Time
}

type TimeRange struct {
	Period string    `json:"period"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
}

// Revenue is the top of the statement. GrossRevenue is the sum of sale totals
// (what customers paid). DiscountAmount/VATAmount are informational sub-lines.
// Refunds is reserved (returns are not tracked yet) and is always 0.
type Revenue struct {
	SalesCount     int64   `json:"sales_count"`
	GrossRevenue   float64 `json:"gross_revenue"`
	DiscountAmount float64 `json:"discount_amount"`
	VATAmount      float64 `json:"vat_amount"`
	Refunds        float64 `json:"refunds"`
}

// COGS = SUM(sale_item.quantity * COALESCE(unit_cost, product.cost_price, 0)) over
// the window. New sales freeze the sale-time cost in sale_items.unit_cost
// (migration 025); historical rows (NULL unit_cost) fall back to the product's
// current cost. MissingCostLines counts sold lines with no resolvable cost (or a
// deleted product) — these understate COGS and overstate profit, surfaced as a
// data-quality warning.
type COGS struct {
	Total            float64 `json:"total"`
	MissingCostLines int64   `json:"missing_cost_lines"`
}

type CategoryTotal struct {
	CategoryID string  `json:"category_id"`
	Name       string  `json:"name"`
	Total      float64 `json:"total"`
}

type PaymentMethodStat struct {
	PaymentMethod string  `json:"payment_method"`
	SalesCount    int64   `json:"sales_count"`
	Amount        float64 `json:"amount"`
}

// PnLReport is the full Profit & Loss payload. Derived figures
// (gross_profit, net_profit, margins) are computed by the client from these
// raw components to keep the contract lean.
type PnLReport struct {
	Range             TimeRange           `json:"range"`
	Revenue           Revenue             `json:"revenue"`
	COGS              COGS                `json:"cogs"`
	OperatingExpenses float64             `json:"operating_expenses"`
	ExpenseByCategory []CategoryTotal     `json:"expense_by_category"`
	PaymentBreakdown  []PaymentMethodStat `json:"payment_breakdown"`
}

// ── Executive Summary (Reports → รายงานสรุป) ──────────────────────────────────

// InventorySnapshot is a point-in-time (NOT period-filtered) view of stock health
// over active products. Counts use the same thresholds as the Inventory page:
// out = stock<=0, low = 0<stock<=min_stock, inStock = stock>min_stock.
type InventorySnapshot struct {
	InventoryValue float64 `json:"inventory_value"` // SUM(cost_price*total_stock), stock>0
	InStock        int64   `json:"in_stock"`
	LowStock       int64   `json:"low_stock"`
	OutOfStock     int64   `json:"out_of_stock"`
	MissingCost    int64   `json:"missing_cost"` // active, stock>0, cost<=0
}

type InventoryHealth struct {
	InStock    int64 `json:"in_stock"`
	LowStock   int64 `json:"low_stock"`
	OutOfStock int64 `json:"out_of_stock"`
	DeadStock  int64 `json:"dead_stock"`
}

// DeadStockItem is one idle product in the dead-stock breakdown: its remaining
// units, tied capital at cost, and last sale date (LastSold nil → never sold).
type DeadStockItem struct {
	ProductID   string     `json:"product_id" gorm:"column:product_id"`
	ProductName string     `json:"product_name" gorm:"column:product_name"`
	Remaining   int64      `json:"remaining" gorm:"column:remaining"`
	TiedValue   float64    `json:"tied_value" gorm:"column:tied_value"`
	LastSold    *time.Time `json:"last_sold" gorm:"column:last_sold"`
	NeverSold   bool       `json:"never_sold" gorm:"-"`
}

// DeadStockStat is the dead-stock count + tied capital (at cost) for products whose
// last sale predates `days` ago (or that never sold), plus the per-product breakdown.
// Computed in SQL from full sale history — never from a capped client-side movement scan.
type DeadStockStat struct {
	Days  int             `json:"days"`
	Count int64           `json:"count"`
	Value float64         `json:"value"`
	Items []DeadStockItem `json:"items"`
}

// StockVelocityItem is a per-product snapshot of current stock vs 30-day sales
// velocity. DaysOfStock = CurrentStock / AvgDailySales; nil when no recent sales.
// Top 20 active products ordered by urgency (fewest days first).
type StockVelocityItem struct {
	ProductID     string   `json:"product_id"      gorm:"column:product_id"`
	ProductName   string   `json:"product_name"    gorm:"column:product_name"`
	CurrentStock  int64    `json:"current_stock"   gorm:"column:current_stock"`
	AvgDailySales float64  `json:"avg_daily_sales" gorm:"column:avg_daily_sales"`
	DaysOfStock   *float64 `json:"days_of_stock"   gorm:"column:days_of_stock"`
	MinStock      int64    `json:"min_stock"       gorm:"column:min_stock"`
	MaxStock      int64    `json:"max_stock"       gorm:"column:max_stock"`
	CostPrice     float64  `json:"cost_price"      gorm:"column:cost_price"`
}

// OverstockItem is a product whose current stock exceeds its max_stock threshold.
// CapitalValue = OverstockQty × CostPrice.
type OverstockItem struct {
	ProductID    string  `json:"product_id"    gorm:"column:product_id"`
	ProductName  string  `json:"product_name"  gorm:"column:product_name"`
	CurrentStock int64   `json:"current_stock" gorm:"column:current_stock"`
	MaxStock     int64   `json:"max_stock"     gorm:"column:max_stock"`
	OverstockQty int64   `json:"overstock_qty" gorm:"column:overstock_qty"`
	CostPrice    float64 `json:"cost_price"    gorm:"column:cost_price"`
	CapitalValue float64 `json:"capital_value" gorm:"column:capital_value"`
}

// InventoryReport backs the Inventory Value & Dead Stock report. Both halves are
// DB aggregates (GetInventorySnapshot + GetDeadStock), so the report scales to any
// dataset size with no client-side movement scanning.
type InventoryReport struct {
	Snapshot      InventorySnapshot   `json:"snapshot"`
	DeadStock     DeadStockStat       `json:"dead_stock"`
	StockVelocity []StockVelocityItem `json:"stock_velocity"`
	Overstock     []OverstockItem     `json:"overstock"`
}

// TrendPoint is one day of the sales-performance chart. Profit is the gross
// margin from sales (revenue − COGS); operating expenses are not day-attributable.
type TrendPoint struct {
	Day     string  `json:"day"` // YYYY-MM-DD
	Revenue float64 `json:"revenue"`
	COGS    float64 `json:"cogs"`
	Profit  float64 `json:"profit"`
}

// MonthlyStat is one row of the per-month P&L summary table (Reports → รายงานสรุป).
// Revenue is net of refunds; Profit is gross profit (revenue − COGS). Discount is the
// total bill+item discount granted that month.
type MonthlyStat struct {
	Month    string  `json:"month"` // YYYY-MM
	Orders   int64   `json:"orders"`
	Revenue  float64 `json:"revenue"`
	COGS     float64 `json:"cogs"`
	Profit   float64 `json:"profit"`
	Discount float64 `json:"discount"`
}

type TopProduct struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	QuantitySold int64   `json:"quantity_sold"`
	Revenue      float64 `json:"revenue"`
	// Profit = revenue − (quantity × COALESCE(unit_cost, current cost_price)) —
	// sale-time cost when snapshotted, current product cost otherwise.
	Profit float64 `json:"profit"`
}

// HourStat buckets sales by hour-of-day (0–23, UTC) for the sales-by-hour chart.
type HourStat struct {
	Hour    int     `json:"hour"`
	Revenue float64 `json:"revenue"`
	Orders  int64   `json:"orders"`
}

// ExecutiveSummary is the single-page SALES overview payload (Reports → รายงานสรุป).
// It is intentionally sales/revenue/profit focused — inventory operations
// (stock health, dead stock, low stock) belong to the Inventory Value report and
// are NOT computed here. Derived figures (gross/net profit, AOV) are filled in by
// the service; PreviousRevenue powers the period-over-period growth stat.
type ExecutiveSummary struct {
	Range             TimeRange           `json:"range"`
	Revenue           float64             `json:"revenue"`
	PreviousRevenue   float64             `json:"previous_revenue"`
	Refunds           float64             `json:"refunds"`
	COGS              float64             `json:"cogs"`
	DiscountAmount    float64             `json:"discount_amount"`
	Expenses          float64             `json:"expenses"`
	GrossProfit       float64             `json:"gross_profit"`
	NetProfit         float64             `json:"net_profit"`
	Orders            int64               `json:"orders"`
	ProductsSold      int64               `json:"products_sold"`
	Customers         int64               `json:"customers"`
	AverageOrderValue float64             `json:"average_order_value"`
	SalesTrend        []TrendPoint        `json:"sales_trend"`
	Months            []MonthlyStat       `json:"months"`
	TopProducts       []TopProduct        `json:"top_products"`
	CategoryBreakdown []CategoryTotal     `json:"category_breakdown"`
	PaymentBreakdown  []PaymentMethodStat `json:"payment_breakdown"`
	SalesByHour       []HourStat          `json:"sales_by_hour"`
}
