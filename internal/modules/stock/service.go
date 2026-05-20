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

func (s Service) canManage(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := s.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager"}).
		Count(&count).Error
	return count > 0, err
}

func (s Service) canView(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := s.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ?", storeID, userID).
		Count(&count).Error
	return count > 0, err
}

func (s Service) GetByProduct(ctx context.Context, actor auth.Claims, storeID, productID string) (StockSummary, error) {
	if strings.TrimSpace(storeID) == "" {
		return StockSummary{}, ErrStockForbidden
	}
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return StockSummary{}, err
	}
	if !allowed {
		return StockSummary{}, ErrStockForbidden
	}
	return s.repo.GetByProduct(ctx, storeID, productID)
}

func (s Service) GetByLocation(ctx context.Context, actor auth.Claims, storeID, locationID string) ([]Stock, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, ErrStockForbidden
	}
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrStockForbidden
	}
	return s.repo.GetByLocation(ctx, storeID, locationID)
}

func (s Service) ListLowStock(ctx context.Context, actor auth.Claims, storeID string, threshold int) ([]LowStockItem, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, ErrStockForbidden
	}
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrStockForbidden
	}
	return s.repo.ListLowStock(ctx, storeID, threshold)
}
