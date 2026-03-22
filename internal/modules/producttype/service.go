package producttype

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct{ repo Repository }

func NewService(repo Repository) Service { return Service{repo: repo} }

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateProductTypeRequest) (ProductType, error) {
	if strings.TrimSpace(req.Name) == "" {
		return ProductType{}, ErrInvalidName
	}
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ProductType{}, err
	}
	if !allowed {
		return ProductType{}, ErrForbiddenStoreAccess
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	item := ProductType{
		ID:          newID(),
		StoreID:     storeID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsActive:    isActive,
		CreatedAt:   time.Now().UTC(),
	}
	return s.repo.Create(ctx, item)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]ProductType, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateProductTypeRequest) (ProductType, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ProductType{}, err
	}
	if !allowed {
		return ProductType{}, ErrForbiddenStoreAccess
	}
	item, err := s.repo.GetByID(ctx, storeID, id)
	if err != nil {
		return ProductType{}, err
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if strings.TrimSpace(item.Name) == "" {
		return ProductType{}, ErrInvalidName
	}
	if req.Description != nil {
		item.Description = strings.TrimSpace(*req.Description)
	}
	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}
	item.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, item)
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
