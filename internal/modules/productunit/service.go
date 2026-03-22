package productunit

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service { return Service{repo: repo} }

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateProductUnitRequest) (ProductUnit, error) {
	if strings.TrimSpace(req.Name) == "" {
		return ProductUnit{}, ErrInvalidName
	}
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ProductUnit{}, err
	}
	if !allowed {
		return ProductUnit{}, ErrForbiddenStoreAccess
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	unit := ProductUnit{
		ID:          newID(),
		StoreID:     storeID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsActive:    isActive,
		CreatedAt:   time.Now().UTC(),
	}
	return s.repo.Create(ctx, unit)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]ProductUnit, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateProductUnitRequest) (ProductUnit, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ProductUnit{}, err
	}
	if !allowed {
		return ProductUnit{}, ErrForbiddenStoreAccess
	}
	unit, err := s.repo.GetByID(ctx, storeID, id)
	if err != nil {
		return ProductUnit{}, err
	}
	if req.Name != nil {
		unit.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		unit.Description = strings.TrimSpace(*req.Description)
	}
	if req.IsActive != nil {
		unit.IsActive = *req.IsActive
	}
	if strings.TrimSpace(unit.Name) == "" {
		return ProductUnit{}, ErrInvalidName
	}
	unit.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, unit)
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, id string) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}
	return s.repo.Delete(ctx, storeID, id)
}
