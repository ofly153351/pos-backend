package location

import (
	"context"
	"strings"
	"time"

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

func (s Service) warehouseBelongsToStore(ctx context.Context, storeID, warehouseID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("warehouses").
		Where("id = ? AND store_id = ?", warehouseID, storeID).
		Count(&count).Error
	return count > 0, err
}

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, input CreateLocationRequest) (Location, error) {
	if strings.TrimSpace(storeID) == "" {
		return Location{}, ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Location{}, err
	}
	if !allowed {
		return Location{}, ErrLocationForbidden
	}
	if strings.TrimSpace(input.Name) == "" {
		return Location{}, ErrLocationNameRequired
	}
	ok, err := s.warehouseBelongsToStore(ctx, storeID, input.WarehouseID)
	if err != nil {
		return Location{}, err
	}
	if !ok {
		return Location{}, ErrInvalidWarehouse
	}
	now := time.Now().UTC()
	loc := Location{
		ID:          newID(),
		StoreID:     storeID,
		WarehouseID: input.WarehouseID,
		Name:        strings.TrimSpace(input.Name),
		Code:        strings.TrimSpace(input.Code),
		ZoneName:    strings.TrimSpace(input.ZoneName),
		FloorName:   strings.TrimSpace(input.FloorName),
		IsSalePoint: input.IsSalePoint,
		IsActive:    true,
		CreatedAt:   now,
	}
	return s.repo.Create(ctx, loc)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID, warehouseID string) ([]Location, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, ErrLocationForbidden
	}
	warehouseID = strings.TrimSpace(warehouseID)
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrLocationForbidden
	}
	if warehouseID != "" {
		ok, err := s.warehouseBelongsToStore(ctx, storeID, warehouseID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidWarehouse
		}
	}
	return s.repo.ListByStore(ctx, storeID, warehouseID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, locationID string) (Location, error) {
	if strings.TrimSpace(storeID) == "" {
		return Location{}, ErrLocationForbidden
	}
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Location{}, err
	}
	if !allowed {
		return Location{}, ErrLocationForbidden
	}
	return s.repo.GetByID(ctx, storeID, locationID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, locationID string, input UpdateLocationRequest) (Location, error) {
	if strings.TrimSpace(storeID) == "" {
		return Location{}, ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Location{}, err
	}
	if !allowed {
		return Location{}, ErrLocationForbidden
	}
	current, err := s.repo.GetByID(ctx, storeID, locationID)
	if err != nil {
		return Location{}, err
	}
	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.Code != nil {
		current.Code = strings.TrimSpace(*input.Code)
	}
	if input.ZoneName != nil {
		current.ZoneName = strings.TrimSpace(*input.ZoneName)
	}
	if input.FloorName != nil {
		current.FloorName = strings.TrimSpace(*input.FloorName)
	}
	if input.IsSalePoint != nil {
		current.IsSalePoint = *input.IsSalePoint
	}
	if input.IsActive != nil {
		current.IsActive = *input.IsActive
	}
	current.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, current)
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, locationID string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrLocationForbidden
	}
	return s.repo.Delete(ctx, storeID, locationID)
}
