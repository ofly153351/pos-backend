package dashboard

import "time"

type OverviewQuery struct {
	Period            string
	From              *time.Time
	To                *time.Time
	TopLimit          int
	RecentLimit       int
	LowStockLimit     int
	LowStockThreshold int
}

type Overview struct {
	Range            TimeRange           `json:"range"`
	Summary          Summary             `json:"summary"`
	PaymentBreakdown []PaymentMethodStat `json:"payment_breakdown"`
	TopProducts      []TopProductStat    `json:"top_products"`
	LowStockProducts []LowStockProduct   `json:"low_stock_products"`
	RecentSales      []RecentSale        `json:"recent_sales"`
}

type TimeRange struct {
	Period string    `json:"period"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
}

type Summary struct {
	SalesCount     int64   `json:"sales_count"`
	Revenue        float64 `json:"revenue"`
	TotalItems     int64   `json:"total_items"`
	AverageTicket  float64 `json:"average_ticket"`
	DiscountAmount float64 `json:"discount_amount"`
	VATAmount      float64 `json:"vat_amount"`
}

type PaymentMethodStat struct {
	PaymentMethod string  `json:"payment_method"`
	SalesCount    int64   `json:"sales_count"`
	Amount        float64 `json:"amount"`
}

type TopProductStat struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	QuantitySold int64   `json:"quantity_sold"`
	Amount       float64 `json:"amount"`
}

type LowStockProduct struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku,omitempty"`
	UnitType  string `json:"unit_type"`
	Quantity  int    `json:"quantity"`
}

type RecentSale struct {
	ID            string    `json:"id"`
	SaleNumber    string    `json:"sale_number"`
	TotalItems    int       `json:"total_items"`
	TotalAmount   float64   `json:"total_amount"`
	PaymentMethod string    `json:"payment_method"`
	CashierName   string    `json:"cashier_name,omitempty"`
	CustomerName  string    `json:"customer_name,omitempty"`
	SoldAt        time.Time `json:"sold_at"`
}
