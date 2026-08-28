package producttype

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"pos-backend/internal/modules/auth"
)

type Service struct{ repo Repository }

func NewService(repo Repository) Service { return Service{repo: repo} }

// isDuplicateKey reports whether err is a PostgreSQL unique-constraint violation
// (SQLSTATE 23505) — raised when (store_id, name) already exists.
func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateProductTypeRequest) (ProductType, error) {
	if strings.TrimSpace(req.Name) == "" {
		return ProductType{}, ErrInvalidName
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
	item, err := s.repo.Create(ctx, item)
	if isDuplicateKey(err) {
		return ProductType{}, ErrDuplicateName
	}
	return item, err
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]ProductType, error) {
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateProductTypeRequest) (ProductType, error) {
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
	item, err = s.repo.Update(ctx, item)
	if isDuplicateKey(err) {
		return ProductType{}, ErrDuplicateName
	}
	return item, err
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, id string) error {
	return s.repo.Delete(ctx, storeID, id)
}
