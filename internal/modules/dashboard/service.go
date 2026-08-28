package dashboard

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

const (
	periodToday = "today"
	period7d    = "7d"
	period30d   = "30d"

	defaultTopLimit          = 5
	maxTopLimit              = 20
	defaultRecentLimit       = 10
	maxRecentLimit           = 50
	defaultLowStockLimit     = 10
	maxLowStockLimit         = 50
	defaultLowStockThreshold = 10
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) GetOverview(ctx context.Context, actor auth.Claims, storeID string, query OverviewQuery) (Overview, error) {

	period, from, to, err := normalizeRange(query)
	if err != nil {
		return Overview{}, err
	}
	topLimit, recentLimit, lowStockLimit, lowStockThreshold, err := normalizeLimits(query)
	if err != nil {
		return Overview{}, err
	}

	summary, err := s.repo.GetSummary(ctx, storeID, from, to)
	if err != nil {
		return Overview{}, err
	}
	paymentBreakdown, err := s.repo.GetPaymentBreakdown(ctx, storeID, from, to)
	if err != nil {
		return Overview{}, err
	}
	topProducts, err := s.repo.GetTopProducts(ctx, storeID, from, to, topLimit)
	if err != nil {
		return Overview{}, err
	}
	lowStockProducts, err := s.repo.GetLowStockProducts(ctx, storeID, lowStockThreshold, lowStockLimit)
	if err != nil {
		return Overview{}, err
	}
	recentSales, err := s.repo.GetRecentSales(ctx, storeID, from, to, recentLimit)
	if err != nil {
		return Overview{}, err
	}

	return Overview{
		Range: TimeRange{
			Period: period,
			From:   from,
			To:     to,
		},
		Summary:          summary,
		PaymentBreakdown: paymentBreakdown,
		TopProducts:      topProducts,
		LowStockProducts: lowStockProducts,
		RecentSales:      recentSales,
	}, nil
}

func normalizeRange(query OverviewQuery) (string, time.Time, time.Time, error) {
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
		period = periodToday
	}

	switch period {
	case periodToday:
		y, m, d := now.Date()
		start := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		return period, start, now, nil
	case period7d:
		return period, now.AddDate(0, 0, -7), now, nil
	case period30d:
		return period, now.AddDate(0, 0, -30), now, nil
	default:
		return "", time.Time{}, time.Time{}, ErrInvalidPeriod
	}
}

func normalizeLimits(query OverviewQuery) (int, int, int, int, error) {
	topLimit := query.TopLimit
	if topLimit == 0 {
		topLimit = defaultTopLimit
	}
	recentLimit := query.RecentLimit
	if recentLimit == 0 {
		recentLimit = defaultRecentLimit
	}
	lowStockLimit := query.LowStockLimit
	if lowStockLimit == 0 {
		lowStockLimit = defaultLowStockLimit
	}
	lowStockThreshold := query.LowStockThreshold
	if lowStockThreshold == 0 {
		lowStockThreshold = defaultLowStockThreshold
	}

	if topLimit < 1 || topLimit > maxTopLimit {
		return 0, 0, 0, 0, ErrInvalidLimit
	}
	if recentLimit < 1 || recentLimit > maxRecentLimit {
		return 0, 0, 0, 0, ErrInvalidLimit
	}
	if lowStockLimit < 1 || lowStockLimit > maxLowStockLimit {
		return 0, 0, 0, 0, ErrInvalidLimit
	}
	if lowStockThreshold < 1 || lowStockThreshold > 1000000 {
		return 0, 0, 0, 0, ErrInvalidLimit
	}

	return topLimit, recentLimit, lowStockLimit, lowStockThreshold, nil
}
