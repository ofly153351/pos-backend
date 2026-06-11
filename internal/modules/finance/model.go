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

// COGS = SUM(sale_item.quantity * product.cost_price) over the window. Uses the
// product's CURRENT cost (no historical cost snapshot exists). MissingCostLines
// counts sold lines whose product has no cost (or was deleted) — these
// understate COGS and overstate profit, surfaced as a data-quality warning.
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

// TrendPoint is one day of the sales-performance chart. Profit is the gross
// margin from sales (revenue − COGS); operating expenses are not day-attributable.
type TrendPoint struct {
	Day     string  `json:"day"` // YYYY-MM-DD
	Revenue float64 `json:"revenue"`
	COGS    float64 `json:"cogs"`
	Profit  float64 `json:"profit"`
}

type TopProduct struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	QuantitySold int64   `json:"quantity_sold"`
	Revenue      float64 `json:"revenue"`
	// Profit = revenue − (quantity × current cost_price). Uses current product
	// cost (no per-sale cost snapshot), same caveat as COGS.
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
	COGS              float64             `json:"cogs"`
	Expenses          float64             `json:"expenses"`
	GrossProfit       float64             `json:"gross_profit"`
	NetProfit         float64             `json:"net_profit"`
	Orders            int64               `json:"orders"`
	ProductsSold      int64               `json:"products_sold"`
	Customers         int64               `json:"customers"`
	AverageOrderValue float64             `json:"average_order_value"`
	SalesTrend        []TrendPoint        `json:"sales_trend"`
	TopProducts       []TopProduct        `json:"top_products"`
	CategoryBreakdown []CategoryTotal     `json:"category_breakdown"`
	PaymentBreakdown  []PaymentMethodStat `json:"payment_breakdown"`
	SalesByHour       []HourStat          `json:"sales_by_hour"`
}
