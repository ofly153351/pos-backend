package warehouse_dashboard

import (
	"context"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

// GetDashboard authorises the request then assembles the full dashboard payload.
func (s Service) GetDashboard(ctx context.Context, actor auth.Claims, storeID string, q Query) (DashboardData, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return DashboardData{}, err
	}
	if !allowed {
		return DashboardData{}, ErrForbidden
	}

	from, to := periodBounds(q.Period)

	kpi, err := s.repo.GetKPI(ctx, storeID)
	if err != nil {
		return DashboardData{}, err
	}

	chart, err := s.repo.GetMovementChart(ctx, storeID, from, to)
	if err != nil {
		return DashboardData{}, err
	}

	alerts, err := s.repo.GetLowStockAlerts(ctx, storeID, 10)
	if err != nil {
		return DashboardData{}, err
	}

	sellers, err := s.repo.GetTopSellers(ctx, storeID)
	if err != nil {
		return DashboardData{}, err
	}

	dist, err := s.repo.GetWarehouseDistribution(ctx, storeID)
	if err != nil {
		return DashboardData{}, err
	}

	activity, err := s.repo.GetRecentActivity(ctx, storeID, 20)
	if err != nil {
		return DashboardData{}, err
	}

	return DashboardData{
		KPI:                   kpi,
		MovementChart:         chart,
		LowStockAlerts:        alerts,
		TopSellers:            sellers,
		WarehouseDistribution: dist,
		RecentActivity:        activity,
	}, nil
}

// periodBounds returns [from, to) UTC bounds for the requested period.
func periodBounds(p Period) (time.Time, time.Time) {
	now := time.Now().UTC()
	to := now.Truncate(24*time.Hour).Add(24 * time.Hour) // start of tomorrow
	switch p {
	case Period30d:
		return to.Add(-30 * 24 * time.Hour), to
	case Period3m:
		return to.Add(-90 * 24 * time.Hour), to
	default: // Period7d
		return to.Add(-7 * 24 * time.Hour), to
	}
}
