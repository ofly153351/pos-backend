package warehouse_dashboard

import (
	"context"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	GetKPI(ctx context.Context, storeID string) (KPI, error)
	GetMovementChart(ctx context.Context, storeID string, from, to time.Time) ([]MovementChartPoint, error)
	GetLowStockAlerts(ctx context.Context, storeID string, limit int) ([]LowStockAlert, error)
	GetTopSellers(ctx context.Context, storeID string) ([]TopSeller, error)
	GetWarehouseDistribution(ctx context.Context, storeID string) ([]WarehouseDistribution, error)
	GetRecentActivity(ctx context.Context, storeID string, limit int) ([]RecentActivity, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

// ── KPI ──────────────────────────────────────────────────────────────────────

func (r PostgresRepository) GetKPI(ctx context.Context, storeID string) (KPI, error) {
	var kpi KPI

	// 1. Current stock value
	var stockRow struct {
		StockValue float64 `gorm:"column:stock_value"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(p.cost_price * s.quantity), 0) AS stock_value
		FROM stocks s
		JOIN products p ON p.id = s.product_id
		WHERE s.store_id = ? AND p.deleted_at IS NULL
	`, storeID).Scan(&stockRow).Error; err != nil {
		return kpi, err
	}
	kpi.StockValue = stockRow.StockValue

	// 2. 30-day net movement for trend estimate
	var changeRow struct {
		ReceiveValue float64 `gorm:"column:receive_value"`
		IssueValue   float64 `gorm:"column:issue_value"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN sm.type = 'IN' AND sm.quantity_change > 0
				THEN sm.quantity_change * p.cost_price ELSE 0 END), 0) AS receive_value,
			COALESCE(SUM(CASE WHEN sm.type IN ('OUT','SALE') AND sm.quantity_change < 0
				THEN ABS(sm.quantity_change) * p.cost_price ELSE 0 END), 0) AS issue_value
		FROM stock_movements sm
		JOIN products p ON p.id = sm.product_id
		WHERE sm.store_id = ? AND sm.created_at >= NOW() - INTERVAL '30 days'
	`, storeID).Scan(&changeRow).Error; err != nil {
		return kpi, err
	}
	netChange := changeRow.ReceiveValue - changeRow.IssueValue
	prevValue := kpi.StockValue - netChange
	if prevValue > 0 {
		kpi.StockValueChangePct = math.Round(netChange/prevValue*1000) / 10
	}

	// 3. Total active SKUs + stock action counts
	var skuRow struct {
		TotalSKUs         int64 `gorm:"column:total_skus"`
		AvailableStockQty int64 `gorm:"column:available_stock_qty"`
		LowStockCount     int64 `gorm:"column:low_stock_count"`
		OutOfStockCount   int64 `gorm:"column:out_of_stock_count"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS total_skus,
			COALESCE(SUM(GREATEST(COALESCE(s_agg.qty, 0), 0)), 0) AS available_stock_qty,
			COUNT(*) FILTER (
				WHERE p.min_stock > 0 AND COALESCE(s_agg.qty, 0) <= p.min_stock
			) AS low_stock_count,
			COUNT(*) FILTER (
				WHERE COALESCE(s_agg.qty, 0) = 0
			) AS out_of_stock_count
		FROM products p
		LEFT JOIN (
			SELECT product_id, SUM(quantity) AS qty
			FROM stocks
			GROUP BY product_id
		) s_agg ON s_agg.product_id = p.id
		WHERE p.store_id = ? AND p.is_active = TRUE AND p.deleted_at IS NULL
	`, storeID).Scan(&skuRow).Error; err != nil {
		return kpi, err
	}
	kpi.TotalSKUs = skuRow.TotalSKUs
	kpi.AvailableStockQty = skuRow.AvailableStockQty
	kpi.LowStockCount = skuRow.LowStockCount
	kpi.OutOfStockCount = skuRow.OutOfStockCount

	// 3.5 Pending transfer queue in warehouse inventory
	var transferQueueRow struct {
		PendingTransferRequests int64 `gorm:"column:pending_transfer_requests"`
		InTransitStockQty       int64 `gorm:"column:in_transit_stock_qty"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS pending_transfer_requests,
			COALESCE(SUM(quantity), 0) AS in_transit_stock_qty
		FROM warehouse_inventory
		WHERE store_id = ? AND quantity > 0
	`, storeID).Scan(&transferQueueRow).Error; err != nil {
		return kpi, err
	}
	kpi.PendingTransferRequests = transferQueueRow.PendingTransferRequests
	kpi.InTransitStockQty = transferQueueRow.InTransitStockQty

	// 4. Today's movements (single query)
	var todayRow struct {
		ReceivedQty      int64   `gorm:"column:received_qty"`
		ReceivedValue    float64 `gorm:"column:received_value"`
		IssuedQty        int64   `gorm:"column:issued_qty"`
		IssuedValue      float64 `gorm:"column:issued_value"`
		TransferredQty   int64   `gorm:"column:transferred_qty"`
		TransferredValue float64 `gorm:"column:transferred_value"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN sm.type = 'IN' AND sm.quantity_change > 0
				THEN sm.quantity_change ELSE 0 END), 0) AS received_qty,
			COALESCE(SUM(CASE WHEN sm.type = 'IN' AND sm.quantity_change > 0
				THEN sm.quantity_change * p.cost_price ELSE 0 END), 0) AS received_value,
			COALESCE(SUM(CASE WHEN sm.type IN ('OUT','SALE') AND sm.quantity_change < 0
				THEN ABS(sm.quantity_change) ELSE 0 END), 0) AS issued_qty,
			COALESCE(SUM(CASE WHEN sm.type IN ('OUT','SALE') AND sm.quantity_change < 0
				THEN ABS(sm.quantity_change) * p.cost_price ELSE 0 END), 0) AS issued_value,
			COALESCE(SUM(CASE WHEN sm.type = 'TRANSFER' AND sm.quantity_change > 0
				THEN sm.quantity_change ELSE 0 END), 0) AS transferred_qty,
			COALESCE(SUM(CASE WHEN sm.type = 'TRANSFER' AND sm.quantity_change > 0
				THEN sm.quantity_change * p.cost_price ELSE 0 END), 0) AS transferred_value
		FROM stock_movements sm
		JOIN products p ON p.id = sm.product_id
		WHERE sm.store_id = ?
		  AND sm.created_at >= CURRENT_DATE
		  AND sm.created_at <  CURRENT_DATE + INTERVAL '1 day'
	`, storeID).Scan(&todayRow).Error; err != nil {
		return kpi, err
	}
	kpi.ReceivedTodayQty = todayRow.ReceivedQty
	kpi.ReceivedTodayValue = todayRow.ReceivedValue
	kpi.IssuedTodayQty = todayRow.IssuedQty
	kpi.IssuedTodayValue = todayRow.IssuedValue
	kpi.TransferredTodayQty = todayRow.TransferredQty
	kpi.TransferredTodayValue = todayRow.TransferredValue

	return kpi, nil
}

// ── Movement chart ────────────────────────────────────────────────────────────

func (r PostgresRepository) GetMovementChart(ctx context.Context, storeID string, from, to time.Time) ([]MovementChartPoint, error) {
	type rawRow struct {
		Date          string  `gorm:"column:date"`
		ReceiveValue  float64 `gorm:"column:receive_value"`
		IssueValue    float64 `gorm:"column:issue_value"`
		TransferValue float64 `gorm:"column:transfer_value"`
	}
	var rows []rawRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			DATE(sm.created_at)::text AS date,
			COALESCE(SUM(CASE WHEN sm.type = 'IN' AND sm.quantity_change > 0
				THEN sm.quantity_change * p.cost_price ELSE 0 END), 0) AS receive_value,
			COALESCE(SUM(CASE WHEN sm.type IN ('OUT','SALE') AND sm.quantity_change < 0
				THEN ABS(sm.quantity_change) * p.cost_price ELSE 0 END), 0) AS issue_value,
			COALESCE(SUM(CASE WHEN sm.type = 'TRANSFER' AND sm.quantity_change > 0
				THEN sm.quantity_change * p.cost_price ELSE 0 END), 0) AS transfer_value
		FROM stock_movements sm
		JOIN products p ON p.id = sm.product_id
		WHERE sm.store_id = ? AND sm.created_at >= ? AND sm.created_at < ?
		GROUP BY DATE(sm.created_at)
		ORDER BY date
	`, storeID, from, to).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]MovementChartPoint, len(rows))
	for i, row := range rows {
		out[i] = MovementChartPoint{
			Date:          row.Date,
			ReceiveValue:  row.ReceiveValue,
			IssueValue:    row.IssueValue,
			TransferValue: row.TransferValue,
			TotalValue:    row.ReceiveValue + row.IssueValue + row.TransferValue,
		}
	}
	return out, nil
}

// ── Low-stock alerts ──────────────────────────────────────────────────────────

func (r PostgresRepository) GetLowStockAlerts(ctx context.Context, storeID string, limit int) ([]LowStockAlert, error) {
	type rawRow struct {
		ProductID  string `gorm:"column:product_id"`
		Name       string `gorm:"column:name"`
		SKU        string `gorm:"column:sku"`
		Unit       string `gorm:"column:unit"`
		TotalStock int64  `gorm:"column:total_stock"`
		MinStock   int    `gorm:"column:min_stock"`
	}
	var rows []rawRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			p.id   AS product_id,
			p.name,
			COALESCE(p.sku, '')   AS sku,
			COALESCE(pu.name, '') AS unit,
			p.min_stock,
			COALESCE(SUM(s.quantity), 0) AS total_stock
		FROM products p
		LEFT JOIN stocks s         ON s.product_id  = p.id
		LEFT JOIN product_units pu ON pu.id          = p.product_unit_id
		WHERE p.store_id = ? AND p.is_active = TRUE AND p.min_stock > 0 AND p.deleted_at IS NULL
		GROUP BY p.id, p.name, p.sku, pu.name, p.min_stock
		HAVING COALESCE(SUM(s.quantity), 0) <= p.min_stock
		ORDER BY (COALESCE(SUM(s.quantity), 0)::FLOAT / NULLIF(p.min_stock, 0)) ASC
		LIMIT ?
	`, storeID, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]LowStockAlert, len(rows))
	for i, row := range rows {
		level := "warning"
		if float64(row.TotalStock) <= float64(row.MinStock)*0.2 {
			level = "critical"
		}
		out[i] = LowStockAlert{
			ProductID:  row.ProductID,
			Name:       row.Name,
			SKU:        row.SKU,
			Unit:       row.Unit,
			TotalStock: row.TotalStock,
			MinStock:   row.MinStock,
			AlertLevel: level,
		}
	}
	return out, nil
}

// ── Top sellers ───────────────────────────────────────────────────────────────

func (r PostgresRepository) GetTopSellers(ctx context.Context, storeID string) ([]TopSeller, error) {
	type rawRow struct {
		ProductID   string  `gorm:"column:product_id"`
		Name        string  `gorm:"column:name"`
		Unit        string  `gorm:"column:unit"`
		TodayQty    int64   `gorm:"column:today_qty"`
		WeekQty     int64   `gorm:"column:week_qty"`
		PrevWeekQty int64   `gorm:"column:prev_week_qty"`
		TodayValue  float64 `gorm:"column:today_value"`
	}
	var rows []rawRow
	if err := r.db.WithContext(ctx).Raw(`
		WITH period_sales AS (
			SELECT
				sm.product_id,
				SUM(CASE WHEN sm.created_at >= CURRENT_DATE
					THEN ABS(sm.quantity_change) ELSE 0 END)                                   AS today_qty,
				SUM(CASE WHEN sm.created_at >= NOW() - INTERVAL '7 days'
					THEN ABS(sm.quantity_change) ELSE 0 END)                                   AS week_qty,
				SUM(CASE WHEN sm.created_at >= NOW() - INTERVAL '14 days'
				         AND sm.created_at <  NOW() - INTERVAL '7 days'
					THEN ABS(sm.quantity_change) ELSE 0 END)                                   AS prev_week_qty,
				SUM(CASE WHEN sm.created_at >= CURRENT_DATE
					THEN ABS(sm.quantity_change) * p.base_price ELSE 0 END)                    AS today_value
			FROM stock_movements sm
			JOIN products p ON p.id = sm.product_id
			WHERE sm.store_id = ?
			  AND sm.type IN ('SALE','OUT')
			  AND sm.quantity_change < 0
			  AND sm.created_at >= NOW() - INTERVAL '14 days'
			  AND NOT EXISTS (SELECT 1 FROM credit_sales cs WHERE cs.sale_id = sm.reference_id AND cs.type = 'loan')
			GROUP BY sm.product_id
		)
		SELECT
			ps.product_id,
			p.name,
			COALESCE(pu.name, '') AS unit,
			ps.today_qty,
			ps.week_qty,
			ps.prev_week_qty,
			ps.today_value
		FROM period_sales ps
		JOIN products p        ON p.id  = ps.product_id
		LEFT JOIN product_units pu ON pu.id = p.product_unit_id
		ORDER BY ps.week_qty DESC
		LIMIT 5
	`, storeID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]TopSeller, len(rows))
	for i, row := range rows {
		var trendPct float64
		if row.PrevWeekQty > 0 {
			trendPct = math.Round(float64(row.WeekQty-row.PrevWeekQty)/float64(row.PrevWeekQty)*1000) / 10
		}
		out[i] = TopSeller{
			Rank:       i + 1,
			ProductID:  row.ProductID,
			Name:       row.Name,
			Unit:       row.Unit,
			TodayQty:   row.TodayQty,
			WeekQty:    row.WeekQty,
			TodayValue: row.TodayValue,
			TrendPct:   trendPct,
		}
	}
	return out, nil
}

// ── Warehouse distribution ────────────────────────────────────────────────────

func (r PostgresRepository) GetWarehouseDistribution(ctx context.Context, storeID string) ([]WarehouseDistribution, error) {
	type rawRow struct {
		WarehouseID string  `gorm:"column:warehouse_id"`
		Name        string  `gorm:"column:name"`
		TotalValue  float64 `gorm:"column:total_value"`
	}
	var rows []rawRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			w.id   AS warehouse_id,
			w.name,
			COALESCE(SUM(s.quantity * p.cost_price), 0) AS total_value
		FROM warehouses w
		LEFT JOIN locations l ON l.warehouse_id = w.id
		LEFT JOIN stocks    s ON s.location_id  = l.id
		LEFT JOIN products  p ON p.id           = s.product_id AND p.deleted_at IS NULL
		WHERE w.store_id = ? AND w.is_active = TRUE
		GROUP BY w.id, w.name
		ORDER BY total_value DESC
		LIMIT 10
	`, storeID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	var maxVal float64
	for _, row := range rows {
		if row.TotalValue > maxVal {
			maxVal = row.TotalValue
		}
	}

	out := make([]WarehouseDistribution, len(rows))
	for i, row := range rows {
		fillPct := 0.0
		if maxVal > 0 {
			fillPct = math.Round(row.TotalValue/maxVal*1000) / 10
		}
		out[i] = WarehouseDistribution{
			WarehouseID: row.WarehouseID,
			Name:        row.Name,
			TotalValue:  row.TotalValue,
			FillPct:     fillPct,
		}
	}
	return out, nil
}

// ── Recent activity ───────────────────────────────────────────────────────────

func (r PostgresRepository) GetRecentActivity(ctx context.Context, storeID string, limit int) ([]RecentActivity, error) {
	type rawRow struct {
		ID               string    `gorm:"column:id"`
		Type             string    `gorm:"column:type"`
		QuantityChange   int       `gorm:"column:quantity_change"`
		ProductName      string    `gorm:"column:product_name"`
		Unit             string    `gorm:"column:unit"`
		ReferenceID      *string   `gorm:"column:reference_id"`
		Note             string    `gorm:"column:note"`
		CreatedAt        time.Time `gorm:"column:created_at"`
		LocationName     string    `gorm:"column:location_name"`
		DestLocationName string    `gorm:"column:dest_location_name"`
	}
	var rows []rawRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			sm.id,
			sm.type,
			sm.quantity_change,
			p.name                          AS product_name,
			COALESCE(pu.name, '')           AS unit,
			sm.reference_id,
			sm.note,
			sm.created_at,
			COALESCE(l.name,  '')           AS location_name,
			COALESCE(dl.name, '')           AS dest_location_name
		FROM stock_movements sm
		JOIN products p        ON p.id  = sm.product_id
		LEFT JOIN product_units pu ON pu.id = p.product_unit_id
		LEFT JOIN locations l      ON l.id  = sm.location_id
		LEFT JOIN locations dl     ON dl.id = sm.destination_location_id
		WHERE sm.store_id = ?
		ORDER BY sm.created_at DESC
		LIMIT ?
	`, storeID, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]RecentActivity, len(rows))
	for i, row := range rows {
		refID := ""
		if row.ReferenceID != nil {
			refID = *row.ReferenceID
		}
		out[i] = RecentActivity{
			ID:                  row.ID,
			Type:                row.Type,
			Description:         formatDescription(row.Type, row.ProductName, row.Unit, row.QuantityChange, row.DestLocationName),
			ReferenceID:         refID,
			Time:                row.CreatedAt.Format("15:04"),
			CreatedAt:           row.CreatedAt,
			ProductName:         row.ProductName,
			Unit:                row.Unit,
			QuantityChange:      row.QuantityChange,
			LocationName:        row.LocationName,
			DestinationLocation: row.DestLocationName,
		}
	}

	return out, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func formatDescription(movType, productName, unit string, qty int, destLocation string) string {
	if unit == "" {
		unit = "ชิ้น"
	}
	absQty := qty
	if absQty < 0 {
		absQty = -absQty
	}
	switch movType {
	case "IN":
		return fmt.Sprintf("รับสินค้า %s +%d %s", productName, absQty, unit)
	case "OUT":
		return fmt.Sprintf("จ่ายออก %s -%d %s", productName, absQty, unit)
	case "SALE":
		return fmt.Sprintf("ขายสินค้า %s -%d %s", productName, absQty, unit)
	case "TRANSFER":
		if destLocation != "" {
			return fmt.Sprintf("โอน %s → %s จำนวน %d %s", productName, destLocation, absQty, unit)
		}
		return fmt.Sprintf("โอน %s จำนวน %d %s", productName, absQty, unit)
	case "ADJUST":
		sign := "+"
		if qty < 0 {
			sign = "-"
		}
		return fmt.Sprintf("ปรับสต็อก %s %s%d %s", productName, sign, absQty, unit)
	case "RETURN":
		return fmt.Sprintf("คืนสินค้า %s +%d %s", productName, absQty, unit)
	default:
		return fmt.Sprintf("%s %s %d %s", movType, productName, absQty, unit)
	}
}
