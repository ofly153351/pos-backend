package dashboard

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// saleVoidedStatus mirrors sale.SaleStatusVoided — voided sales (e.g. a cancelled
// credit sale whose goods were restocked) are excluded from every dashboard
// revenue/sales aggregate. Kept local to avoid a module dependency, matching
// finance/repository.go.
const saleVoidedStatus = "voided"

type Repository interface {
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
	GetSummary(ctx context.Context, storeID string, from, to time.Time) (Summary, error)
	GetPaymentBreakdown(ctx context.Context, storeID string, from, to time.Time) ([]PaymentMethodStat, error)
	GetTopProducts(ctx context.Context, storeID string, from, to time.Time, limit int) ([]TopProductStat, error)
	GetLowStockProducts(ctx context.Context, storeID string, threshold, limit int) ([]LowStockProduct, error)
	GetRecentSales(ctx context.Context, storeID string, from, to time.Time, limit int) ([]RecentSale, error)
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

func (r PostgresRepository) GetSummary(ctx context.Context, storeID string, from, to time.Time) (Summary, error) {
	var result Summary
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(`
			COUNT(*) AS sales_count,
			COALESCE(SUM(s.total_amount), 0) AS revenue,
			COALESCE(SUM(s.total_items), 0) AS total_items,
			COALESCE(ROUND(AVG(s.total_amount)::numeric, 2), 0) AS average_ticket,
			COALESCE(SUM(s.discount_amount), 0) AS discount_amount,
			COALESCE(SUM(s.vat_amount), 0) AS vat_amount
		`).
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ?", storeID, from, to, saleVoidedStatus).
		Scan(&result).Error
	return result, err
}

func (r PostgresRepository) GetPaymentBreakdown(ctx context.Context, storeID string, from, to time.Time) ([]PaymentMethodStat, error) {
	var items []PaymentMethodStat
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(`
			COALESCE(NULLIF(TRIM(s.payment_method), ''), 'unknown') AS payment_method,
			COUNT(*) AS sales_count,
			COALESCE(SUM(s.total_amount), 0) AS amount
		`).
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ?", storeID, from, to, saleVoidedStatus).
		Group("COALESCE(NULLIF(TRIM(s.payment_method), ''), 'unknown')").
		Order("amount DESC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetTopProducts(ctx context.Context, storeID string, from, to time.Time, limit int) ([]TopProductStat, error) {
	var items []TopProductStat
	err := r.db.WithContext(ctx).
		Table("sale_items si").
		Select(`
			si.product_id,
			si.product_name,
			COALESCE(SUM(si.quantity), 0) AS quantity_sold,
			COALESCE(SUM(si.line_total), 0) AS amount
		`).
		Joins("JOIN sales s ON s.id = si.sale_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ?", storeID, from, to, saleVoidedStatus).
		Group("si.product_id, si.product_name").
		Order("quantity_sold DESC, amount DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

// GetLowStockProducts returns active products whose operational on-hand (ready_stock —
// the sale-point total) is at or below their effective minimum. The effective minimum is
// the product's own min_stock when set (> 0), otherwise the supplied fallback threshold
// (the dashboard passes 10). `quantity` is ready_stock so the caller can split the list:
// out-of-stock = quantity <= 0, low-stock = quantity > 0. This replaces the previous
// unconditional fixed threshold so the Global Dashboard matches the Inventory/Warehouse
// pages (which already use min_stock).
func (r PostgresRepository) GetLowStockProducts(ctx context.Context, storeID string, threshold, limit int) ([]LowStockProduct, error) {
	var items []LowStockProduct
	err := r.db.WithContext(ctx).
		Table("product_view pv").
		Select("pv.id AS product_id, pv.name, COALESCE(pv.sku, '') AS sku, COALESCE(pv.product_unit_name, '') AS unit_type, pv.min_stock, pv.max_stock, COALESCE(pv.ready_stock, 0) AS quantity").
		Where("pv.store_id = ? AND pv.is_active = TRUE AND pv.deleted_at IS NULL AND COALESCE(pv.ready_stock, 0) <= CASE WHEN COALESCE(pv.min_stock, 0) > 0 THEN pv.min_stock ELSE ? END", storeID, threshold).
		Order("COALESCE(pv.ready_stock, 0) ASC, pv.updated_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetRecentSales(ctx context.Context, storeID string, from, to time.Time, limit int) ([]RecentSale, error) {
	var items []RecentSale
	err := r.db.WithContext(ctx).
		Table("sales s").
		Select(`
			s.id,
			s.sale_number,
			s.total_items,
			s.total_amount,
			COALESCE(NULLIF(TRIM(s.payment_method), ''), 'unknown') AS payment_method,
			COALESCE(u.full_name, '') AS cashier_name,
			COALESCE(c.full_name, '') AS customer_name,
			s.sold_at
		`).
		Joins("LEFT JOIN users u ON u.id = s.cashier_user_id").
		Joins("LEFT JOIN customers c ON c.id = s.customer_id").
		Where("s.store_id = ? AND s.sold_at >= ? AND s.sold_at < ? AND s.status <> ?", storeID, from, to, saleVoidedStatus).
		Order("s.sold_at DESC, s.created_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}
