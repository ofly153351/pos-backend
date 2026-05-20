package warehouse

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct{ repo Repository }

func NewService(repo Repository) Service { return Service{repo: repo} }

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateWarehouseRequest) (Warehouse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return Warehouse{}, ErrInvalidName
	}
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Warehouse{}, err
	}
	if !allowed {
		return Warehouse{}, ErrForbiddenStoreAccess
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	item := Warehouse{
		ID:          newID(),
		StoreID:     storeID,
		Name:        strings.TrimSpace(req.Name),
		Code:        strings.TrimSpace(req.Code),
		Address:     strings.TrimSpace(req.Address),
		Phone:       strings.TrimSpace(req.Phone),
		ContactName: strings.TrimSpace(req.ContactName),
		IsActive:    isActive,
		CreatedAt:   time.Now().UTC(),
	}
	return s.repo.Create(ctx, item)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]Warehouse, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, id string) (Warehouse, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Warehouse{}, err
	}
	if !allowed {
		return Warehouse{}, ErrForbiddenStoreAccess
	}
	return s.repo.GetByID(ctx, storeID, id)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateWarehouseRequest) (Warehouse, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Warehouse{}, err
	}
	if !allowed {
		return Warehouse{}, ErrForbiddenStoreAccess
	}
	item, err := s.repo.GetByID(ctx, storeID, id)
	if err != nil {
		return Warehouse{}, err
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if strings.TrimSpace(item.Name) == "" {
		return Warehouse{}, ErrInvalidName
	}
	if req.Code != nil {
		item.Code = strings.TrimSpace(*req.Code)
	}
	if req.Address != nil {
		item.Address = strings.TrimSpace(*req.Address)
	}
	if req.Phone != nil {
		item.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.ContactName != nil {
		item.ContactName = strings.TrimSpace(*req.ContactName)
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

// ──────────────────────────────────────────────
// Warehouse-Product service methods
// ──────────────────────────────────────────────

func (s Service) AddProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID string, req AddWarehouseProductRequest) (WarehouseProduct, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return WarehouseProduct{}, err
	}
	if !allowed {
		return WarehouseProduct{}, ErrForbiddenStoreAccess
	}

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return WarehouseProduct{}, err
	}

	// Verify product belongs to store
	if _, err := s.repo.ProductBelongsToStore(ctx, storeID, req.ProductID); err != nil {
		return WarehouseProduct{}, err
	}

	return s.repo.AddProduct(ctx, storeID, warehouseID, req.ProductID, req.Quantity)
}

func (s Service) ListProducts(ctx context.Context, actor auth.Claims, storeID, warehouseID string) ([]WarehouseProduct, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return nil, err
	}

	return s.repo.ListProducts(ctx, warehouseID)
}

func (s Service) UpdateProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID, productID string, quantity int) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return err
	}

	// Verify product exists in warehouse
	exists, err := s.repo.ProductExistsInWarehouse(ctx, warehouseID, productID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProductNotInWarehouse
	}

	return s.repo.UpdateProduct(ctx, storeID, warehouseID, productID, quantity)
}

func (s Service) RemoveProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID, productID string) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return err
	}

	// Verify product exists in warehouse
	exists, err := s.repo.ProductExistsInWarehouse(ctx, warehouseID, productID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProductNotInWarehouse
	}

	return s.repo.RemoveProduct(ctx, storeID, warehouseID, productID)
}
