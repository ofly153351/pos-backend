package stock

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
	db   *gorm.DB
}

func NewService(repo Repository, db *gorm.DB) Service {
	return Service{repo: repo, db: db}
}

func (s Service) GetByProduct(ctx context.Context, actor auth.Claims, storeID, productID string) (StockSummary, error) {
	if strings.TrimSpace(storeID) == "" {
		return StockSummary{}, ErrStockForbidden
	}
	return s.repo.GetByProduct(ctx, storeID, productID)
}

func (s Service) GetByLocation(ctx context.Context, actor auth.Claims, storeID, locationID string) ([]Stock, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, ErrStockForbidden
	}
	return s.repo.GetByLocation(ctx, storeID, locationID)
}

func (s Service) ListLowStock(ctx context.Context, actor auth.Claims, storeID string, threshold int) ([]LowStockItem, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, ErrStockForbidden
	}
	return s.repo.ListLowStock(ctx, storeID, threshold)
}
