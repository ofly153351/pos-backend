package finance

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

const (
	period7d  = "7d"
	period30d = "30d"
	period90d = "90d"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

// GetPnL composes the Profit & Loss report for a store over the requested window.
// Raw revenue/COGS/expense components are returned; the client derives gross
// profit, net profit and the margin ratios.
func (s Service) GetPnL(ctx context.Context, actor auth.Claims, storeID string, query PnLQuery) (PnLReport, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return PnLReport{}, err
	}
	if !allowed {
		return PnLReport{}, ErrForbiddenStoreAccess
	}

	period, from, to, err := normalizeRange(query)
	if err != nil {
		return PnLReport{}, err
	}

	revenue, err := s.repo.GetRevenue(ctx, storeID, from, to)
	if err != nil {
		return PnLReport{}, err
	}
	cogs, err := s.repo.GetCOGS(ctx, storeID, from, to)
	if err != nil {
		return PnLReport{}, err
	}
	operatingExpenses, err := s.repo.GetOperatingExpenses(ctx, storeID, from, to)
	if err != nil {
		return PnLReport{}, err
	}
	expenseByCategory, err := s.repo.GetExpenseByCategory(ctx, storeID, from, to)
	if err != nil {
		return PnLReport{}, err
	}
	paymentBreakdown, err := s.repo.GetPaymentBreakdown(ctx, storeID, from, to)
	if err != nil {
		return PnLReport{}, err
	}

	return PnLReport{
		Range:             TimeRange{Period: period, From: from, To: to},
		Revenue:           revenue,
		COGS:              cogs,
		OperatingExpenses: operatingExpenses,
		ExpenseByCategory: expenseByCategory,
		PaymentBreakdown:  paymentBreakdown,
	}, nil
}

const (
	topProductsLimit = 10
	deadStockDays    = 90
)

// GetSummary composes the single-page executive overview for a store. Revenue/
// COGS/expense components are summed in SQL and reused from the P&L queries;
// inventory figures are a current snapshot (not period-filtered).
func (s Service) GetSummary(ctx context.Context, actor auth.Claims, storeID string, query PnLQuery) (ExecutiveSummary, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	if !allowed {
		return ExecutiveSummary{}, ErrForbiddenStoreAccess
	}

	period, from, to, err := normalizeRange(query)
	if err != nil {
		return ExecutiveSummary{}, err
	}

	revenue, err := s.repo.GetRevenue(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	cogs, err := s.repo.GetCOGS(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	expenses, err := s.repo.GetOperatingExpenses(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	// Previous equal-length window → period-over-period revenue growth.
	prevTo := from
	prevFrom := from.Add(-to.Sub(from))
	prevRevenue, err := s.repo.GetRevenue(ctx, storeID, prevFrom, prevTo)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	units, customers, err := s.repo.GetSalesCounters(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	topProducts, err := s.repo.GetTopProducts(ctx, storeID, from, to, topProductsLimit)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	categoryBreakdown, err := s.repo.GetCategoryBreakdown(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	salesTrend, orders, err := s.repo.GetSalesTrend(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	paymentBreakdown, err := s.repo.GetPaymentBreakdown(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}
	salesByHour, err := s.repo.GetSalesByHour(ctx, storeID, from, to)
	if err != nil {
		return ExecutiveSummary{}, err
	}

	netRevenue := revenue.GrossRevenue - revenue.Refunds
	prevNetRevenue := prevRevenue.GrossRevenue - prevRevenue.Refunds
	grossProfit := netRevenue - cogs.Total
	netProfit := grossProfit - expenses
	var aov float64
	if orders > 0 {
		aov = netRevenue / float64(orders)
	}

	return ExecutiveSummary{
		Range:             TimeRange{Period: period, From: from, To: to},
		Revenue:           netRevenue,
		PreviousRevenue:   prevNetRevenue,
		Refunds:           revenue.Refunds,
		COGS:              cogs.Total,
		Expenses:          expenses,
		GrossProfit:       grossProfit,
		NetProfit:         netProfit,
		Orders:            orders,
		ProductsSold:      units,
		Customers:         customers,
		AverageOrderValue: aov,
		SalesTrend:        salesTrend,
		TopProducts:       topProducts,
		CategoryBreakdown: categoryBreakdown,
		PaymentBreakdown:  paymentBreakdown,
		SalesByHour:       salesByHour,
	}, nil
}

// GetInventoryReport returns the point-in-time stock-health snapshot plus the
// dead-stock count/value for the given idle threshold (default 30 days). Both are
// SQL aggregates over the full dataset — no capped client-side movement scan.
func (s Service) GetInventoryReport(ctx context.Context, actor auth.Claims, storeID string, deadDays int) (InventoryReport, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return InventoryReport{}, err
	}
	if !allowed {
		return InventoryReport{}, ErrForbiddenStoreAccess
	}

	if deadDays <= 0 {
		deadDays = 30
	}

	snapshot, err := s.repo.GetInventorySnapshot(ctx, storeID)
	if err != nil {
		return InventoryReport{}, err
	}

	soldBefore := time.Now().UTC().AddDate(0, 0, -deadDays)
	deadItems, count, value, err := s.repo.GetDeadStock(ctx, storeID, soldBefore)
	if err != nil {
		return InventoryReport{}, err
	}

	velocity, err := s.repo.GetStockVelocity(ctx, storeID)
	if err != nil {
		return InventoryReport{}, err
	}

	overstock, err := s.repo.GetOverstockItems(ctx, storeID)
	if err != nil {
		return InventoryReport{}, err
	}

	return InventoryReport{
		Snapshot:      snapshot,
		DeadStock:     DeadStockStat{Days: deadDays, Count: count, Value: value, Items: deadItems},
		StockVelocity: velocity,
		Overstock:     overstock,
	}, nil
}

// normalizeRange resolves either an explicit From/To window (max 366 days) or a
// named period (7d/30d/90d, default 30d) into a concrete [from, to) range.
func normalizeRange(query PnLQuery) (string, time.Time, time.Time, error) {
	now := time.Now().UTC()

	if query.From != nil || query.To != nil {
		if query.From == nil || query.To == nil {
			return "custom", time.Time{}, time.Time{}, ErrInvalidTimeRange
		}
		from := query.From.UTC()
		to := query.To.UTC()
		if !from.Before(to) {
			return "custom", time.Time{}, time.Time{}, ErrInvalidTimeRange
		}
		if to.Sub(from) > 366*24*time.Hour {
			return "custom", time.Time{}, time.Time{}, ErrInvalidTimeRange
		}
		return "custom", from, to, nil
	}

	period := strings.ToLower(strings.TrimSpace(query.Period))
	if period == "" {
		period = period30d
	}

	switch period {
	case period7d:
		return period, now.AddDate(0, 0, -7), now, nil
	case period30d:
		return period, now.AddDate(0, 0, -30), now, nil
	case period90d:
		return period, now.AddDate(0, 0, -90), now, nil
	default:
		return "", time.Time{}, time.Time{}, ErrInvalidPeriod
	}
}
