package product

import (
	"context"
	"math"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo    Repository
	storage ImageStorage
}

func NewService(repo Repository, storage ImageStorage) Service {
	return Service{repo: repo, storage: storage}
}

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, input CreateProductRequest) (Product, error) {
	if err := validateCreate(input); err != nil {
		return Product{}, err
	}

	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Product{}, err
	}
	if !allowed {
		return Product{}, ErrForbiddenStoreAccess
	}

	ok, err := s.repo.ProductTypeExists(ctx, storeID, strings.TrimSpace(input.ProductTypeID))
	if err != nil {
		return Product{}, err
	}
	if !ok {
		return Product{}, ErrInvalidProductTypeID
	}
	ok, err = s.repo.ProductUnitExists(ctx, storeID, strings.TrimSpace(input.ProductUnitID))
	if err != nil {
		return Product{}, err
	}
	if !ok {
		return Product{}, ErrInvalidProductUnitID
	}

	imageURL, err := s.storage.SaveProductImage(input.ImageFile)
	if err != nil {
		return Product{}, err
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	product := Product{
		ID:                  newID(),
		StoreID:             storeID,
		ProductTypeID:       strings.TrimSpace(input.ProductTypeID),
		Name:                strings.TrimSpace(input.Name),
		SKU:                 strings.TrimSpace(input.SKU),
		ProductUnitID:       strings.TrimSpace(input.ProductUnitID),
		ImageURL:            imageURL,
		Quantity:            0,
		BasePrice:           input.BasePrice,
		SpecialPrice:        input.SpecialPrice,
		SpecialPriceStartAt: input.SpecialPriceStartAt,
		SpecialPriceEndAt:   input.SpecialPriceEndAt,
		IsActive:            isActive,
		CreatedAt:           time.Now().UTC(),
	}
	if input.Quantity != nil {
		product.Quantity = *input.Quantity
	}

	return s.repo.Create(ctx, product)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string, query ListProductsQuery) (ProductListResult, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ProductListResult{}, err
	}
	if !allowed {
		return ProductListResult{}, ErrForbiddenStoreAccess
	}
	if query.Page < 1 || query.Limit < 1 {
		return ProductListResult{}, ErrInvalidPagination
	}

	products, total, err := s.repo.ListByStore(ctx, storeID, query.Page, query.Limit)
	if err != nil {
		return ProductListResult{}, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return ProductListResult{
		Items:      products,
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    query.Page < totalPages,
		HasPrev:    query.Page > 1,
	}, nil
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, productID string) (Product, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Product{}, err
	}
	if !allowed {
		return Product{}, ErrForbiddenStoreAccess
	}
	return s.repo.GetByID(ctx, storeID, productID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, productID string, input UpdateProductRequest) (Product, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Product{}, err
	}
	if !allowed {
		return Product{}, ErrForbiddenStoreAccess
	}

	current, err := s.repo.GetByID(ctx, storeID, productID)
	if err != nil {
		return Product{}, err
	}

	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.SKU != nil {
		current.SKU = strings.TrimSpace(*input.SKU)
	}
	if input.ProductTypeID != nil {
		current.ProductTypeID = strings.TrimSpace(*input.ProductTypeID)
	}
	if input.ProductUnitID != nil {
		current.ProductUnitID = strings.TrimSpace(*input.ProductUnitID)
	}
	if input.BasePrice != nil {
		current.BasePrice = *input.BasePrice
	}
	if input.Quantity != nil {
		current.Quantity = *input.Quantity
	}
	if input.IsActive != nil {
		current.IsActive = *input.IsActive
	}
	if input.ClearSpecialPrice {
		current.SpecialPrice = nil
	}
	if input.SpecialPrice != nil {
		current.SpecialPrice = input.SpecialPrice
	}
	if input.ClearSpecialWindow {
		current.SpecialPriceStartAt = nil
		current.SpecialPriceEndAt = nil
	}
	if input.SpecialPriceStartAt != nil {
		current.SpecialPriceStartAt = input.SpecialPriceStartAt
	}
	if input.SpecialPriceEndAt != nil {
		current.SpecialPriceEndAt = input.SpecialPriceEndAt
	}
	if input.ImageFile != nil {
		imageURL, err := s.storage.SaveProductImage(input.ImageFile)
		if err != nil {
			return Product{}, err
		}
		current.ImageURL = imageURL
	}

	if err := validateExisting(current); err != nil {
		return Product{}, err
	}
	ok, err := s.repo.ProductTypeExists(ctx, storeID, strings.TrimSpace(current.ProductTypeID))
	if err != nil {
		return Product{}, err
	}
	if !ok {
		return Product{}, ErrInvalidProductTypeID
	}
	ok, err = s.repo.ProductUnitExists(ctx, storeID, strings.TrimSpace(current.ProductUnitID))
	if err != nil {
		return Product{}, err
	}
	if !ok {
		return Product{}, ErrInvalidProductUnitID
	}

	current.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, current)
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, productID string) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}
	return s.repo.Delete(ctx, storeID, productID)
}

func resolveEffectivePrice(product Product, now time.Time) float64 {
	if product.SpecialPrice == nil {
		return product.BasePrice
	}
	if product.SpecialPriceStartAt != nil && now.Before(*product.SpecialPriceStartAt) {
		return product.BasePrice
	}
	if product.SpecialPriceEndAt != nil && now.After(*product.SpecialPriceEndAt) {
		return product.BasePrice
	}
	return *product.SpecialPrice
}

func validateCreate(input CreateProductRequest) error {
	product := Product{
		Name:                strings.TrimSpace(input.Name),
		ProductUnitID:       strings.TrimSpace(input.ProductUnitID),
		Quantity:            0,
		BasePrice:           input.BasePrice,
		SpecialPrice:        input.SpecialPrice,
		SpecialPriceStartAt: input.SpecialPriceStartAt,
		SpecialPriceEndAt:   input.SpecialPriceEndAt,
	}
	if input.Quantity != nil {
		product.Quantity = *input.Quantity
	}
	return validateExisting(product)
}

func validateExisting(product Product) error {
	if strings.TrimSpace(product.Name) == "" {
		return ErrInvalidProductName
	}
	if strings.TrimSpace(product.ProductUnitID) == "" {
		return ErrInvalidProductUnitID
	}
	if product.Quantity < 0 {
		return ErrInvalidQuantity
	}
	if product.BasePrice < 0 {
		return ErrInvalidBasePrice
	}
	if product.SpecialPrice != nil && *product.SpecialPrice > product.BasePrice {
		return ErrInvalidSpecialPrice
	}
	if product.SpecialPriceStartAt != nil && product.SpecialPriceEndAt != nil && product.SpecialPriceEndAt.Before(*product.SpecialPriceStartAt) {
		return ErrInvalidSpecialPriceDate
	}
	return nil
}
