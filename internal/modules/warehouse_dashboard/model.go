package warehouse_dashboard

import "time"

// Period constants for the dashboard query.
type Period string

const (
	Period7d  Period = "7d"
	Period30d Period = "30d"
	Period3m  Period = "3m"
)

// Query holds the parsed request parameters.
type Query struct {
	Period Period
}

// KPI is the top-row summary card data.
type KPI struct {
	StockValue            float64 `json:"stock_value"`
	StockValueChangePct   float64 `json:"stock_value_change_pct"`
	TotalSKUs             int64   `json:"total_skus"`
	LowStockCount         int64   `json:"low_stock_count"`
	ReceivedTodayQty      int64   `json:"received_today_qty"`
	ReceivedTodayValue    float64 `json:"received_today_value"`
	IssuedTodayQty        int64   `json:"issued_today_qty"`
	IssuedTodayValue      float64 `json:"issued_today_value"`
	TransferredTodayQty   int64   `json:"transferred_today_qty"`
	TransferredTodayValue float64 `json:"transferred_today_value"`
}

// MovementChartPoint is a single day's movement aggregation.
type MovementChartPoint struct {
	Date          string  `json:"date"`           // ISO date "2026-05-18"
	ReceiveValue  float64 `json:"receive_value"`  // cost_price × qty received
	IssueValue    float64 `json:"issue_value"`    // cost_price × qty issued
	TransferValue float64 `json:"transfer_value"` // cost_price × qty transferred
	TotalValue    float64 `json:"total_value"`    // sum of the three
}

// LowStockAlert is one product below its reorder point.
type LowStockAlert struct {
	ProductID  string `json:"product_id"`
	Name       string `json:"name"`
	SKU        string `json:"sku"`
	Unit       string `json:"unit"`
	TotalStock int64  `json:"total_stock"`
	MinStock   int    `json:"min_stock"`
	// "critical" when total_stock ≤ min_stock × 0.2, otherwise "warning".
	AlertLevel string `json:"alert_level"`
}

// TopSeller is one of the five best-selling products.
type TopSeller struct {
	Rank       int     `json:"rank"`
	ProductID  string  `json:"product_id"`
	Name       string  `json:"name"`
	Unit       string  `json:"unit"`
	TodayQty   int64   `json:"today_qty"`
	WeekQty    int64   `json:"week_qty"`
	TodayValue float64 `json:"today_value"`
	// Percentage change vs previous 7-day window; can be negative.
	TrendPct float64 `json:"trend_pct"`
}

// WarehouseDistribution shows stock value per warehouse.
type WarehouseDistribution struct {
	WarehouseID string  `json:"warehouse_id"`
	Name        string  `json:"name"`
	TotalValue  float64 `json:"total_value"`
	// 0–100 percentage relative to the warehouse with the highest value.
	FillPct float64 `json:"fill_pct"`
}

// RecentActivity is a single stock-movement event.
type RecentActivity struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // IN | OUT | SALE | TRANSFER | ADJUST | RETURN
	Description string    `json:"description"` // human-readable Thai string
	ReferenceID string    `json:"reference_id"`
	Time        string    `json:"time"` // "HH:MM"
	CreatedAt   time.Time `json:"created_at"`
}

// DashboardData is the complete response body.
type DashboardData struct {
	KPI                   KPI                     `json:"kpi"`
	MovementChart         []MovementChartPoint    `json:"movement_chart"`
	LowStockAlerts        []LowStockAlert         `json:"low_stock_alerts"`
	TopSellers            []TopSeller             `json:"top_sellers"`
	WarehouseDistribution []WarehouseDistribution `json:"warehouse_distribution"`
	RecentActivity        []RecentActivity        `json:"recent_activity"`
}
