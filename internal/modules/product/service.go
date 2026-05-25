package product

import (
	"context"
	"log/slog"
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
	ok, err = s.repo.BrandExists(ctx, storeID, strings.TrimSpace(input.BrandID))
	if err != nil {
		return Product{}, err
	}
	if !ok {
		return Product{}, ErrInvalidBrandID
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
		BrandID:             strings.TrimSpace(input.BrandID),
		SKU:                 strings.TrimSpace(input.SKU),
		Barcode:             strings.TrimSpace(input.Barcode),
		ProductCode:         strings.TrimSpace(input.ProductCode),
		Description:         strings.TrimSpace(input.Description),
		StorageLocation:     strings.TrimSpace(input.StorageLocation),
		ProductUnitID:       strings.TrimSpace(input.ProductUnitID),
		ImageURL:            imageURL,
		BasePrice:           input.BasePrice,
		CostPrice:           input.CostPrice,
		SpecialPrice:        input.SpecialPrice,
		SpecialPriceStartAt: input.SpecialPriceStartAt,
		SpecialPriceEndAt:   input.SpecialPriceEndAt,
		IsActive:            isActive,
		CreatedAt:           time.Now().UTC(),
	}
	if input.MinStock != nil {
		product.MinStock = *input.MinStock
	}
	if input.MaxStock != nil {
		product.MaxStock = input.MaxStock
	}
	if product.SKU == "" {
		sku, err := s.generateUniqueSKU(ctx, storeID)
		if err != nil {
			return Product{}, err
		}
		product.SKU = sku
	}
	if product.Barcode == "" {
		barcode, err := s.generateUniqueBarcode(ctx, storeID)
		if err != nil {
			return Product{}, err
		}
		product.Barcode = barcode
	}

	created, err := s.repo.Create(ctx, product)
	if err != nil {
		if imageURL != "" {
			if deleteErr := s.storage.DeleteProductImage(imageURL); deleteErr != nil {
				slog.WarnContext(ctx, "failed to delete newly saved product image after database create failure",
					"store_id", storeID,
					"product_id", product.ID,
					"image_url", imageURL,
					"delete_error", deleteErr.Error(),
					"create_error", err.Error(),
				)
			}
		}
		return Product{}, err
	}
	return created, nil
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

	if query.StockStatus != "" && query.StockStatus != "low_stock" && query.StockStatus != "out_of_stock" {
		return ProductListResult{}, ErrInvalidStockStatus
	}

	products, total, err := s.repo.ListByStore(ctx, storeID, query.Page, query.Limit, query.StockStatus)
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
	if input.BrandID != nil {
		current.BrandID = strings.TrimSpace(*input.BrandID)
	}
	if input.SKU != nil {
		current.SKU = strings.TrimSpace(*input.SKU)
	}
	if input.ClearSKU {
		current.SKU = ""
	}
	if input.Barcode != nil {
		current.Barcode = strings.TrimSpace(*input.Barcode)
	}
	if input.ClearBarcode {
		current.Barcode = ""
	}
	if input.ProductCode != nil {
		current.ProductCode = strings.TrimSpace(*input.ProductCode)
	}
	if input.ClearProductCode {
		current.ProductCode = ""
	}
	if input.Description != nil {
		current.Description = strings.TrimSpace(*input.Description)
	}
	if input.StorageLocation != nil {
		current.StorageLocation = strings.TrimSpace(*input.StorageLocation)
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
	if input.CostPrice != nil {
		current.CostPrice = *input.CostPrice
	}
	if input.MinStock != nil {
		current.MinStock = *input.MinStock
	}
	if input.MaxStock != nil {
		current.MaxStock = input.MaxStock
	}
	if input.ClearMaxStock {
		current.MaxStock = nil
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
	oldImageURL := current.ImageURL
	newImageURL := ""
	if input.ImageFile != nil {
		imageURL, err := s.storage.SaveProductImage(input.ImageFile)
		if err != nil {
			return Product{}, err
		}
		newImageURL = imageURL
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
	ok, err = s.repo.BrandExists(ctx, storeID, strings.TrimSpace(current.BrandID))
	if err != nil {
		return Product{}, err
	}
	if !ok {
		return Product{}, ErrInvalidBrandID
	}

	current.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		if newImageURL != "" {
			if deleteErr := s.storage.DeleteProductImage(newImageURL); deleteErr != nil {
				slog.WarnContext(ctx, "failed to delete newly saved product image after database update failure",
					"store_id", storeID,
					"product_id", productID,
					"new_image_url", newImageURL,
					"delete_error", deleteErr.Error(),
					"update_error", err.Error(),
				)
			}
		}
		return Product{}, err
	}
	if newImageURL != "" && oldImageURL != "" && oldImageURL != newImageURL {
		if deleteErr := s.storage.DeleteProductImage(oldImageURL); deleteErr != nil {
			slog.WarnContext(ctx, "failed to delete replaced product image",
				"store_id", storeID,
				"product_id", productID,
				"old_image_url", oldImageURL,
				"new_image_url", newImageURL,
				"delete_error", deleteErr.Error(),
			)
		}
	}
	return updated, nil
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, productID string) error {
	if strings.TrimSpace(storeID) == "" || strings.TrimSpace(productID) == "" {
		return ErrProductNotFound
	}
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}

	// Verify product exists and belongs to store
	exists, err := s.repo.ExistsByID(ctx, storeID, productID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProductNotFound
	}

	return s.repo.SoftDelete(ctx, storeID, productID)
}

func (s Service) GenerateMissingSKU(ctx context.Context, actor auth.Claims, storeID string) (GenerateMissingSKUResult, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return GenerateMissingSKUResult{}, err
	}
	if !allowed {
		return GenerateMissingSKUResult{}, ErrForbiddenStoreAccess
	}

	ids, err := s.repo.ListProductIDsWithoutSKU(ctx, storeID)
	if err != nil {
		return GenerateMissingSKUResult{}, err
	}

	updated := 0
	for _, productID := range ids {
		sku, err := s.generateUniqueSKU(ctx, storeID)
		if err != nil {
			return GenerateMissingSKUResult{}, err
		}
		if err := s.repo.UpdateSKU(ctx, storeID, productID, sku, time.Now().UTC()); err != nil {
			return GenerateMissingSKUResult{}, err
		}
		updated++
	}

	return GenerateMissingSKUResult{UpdatedCount: updated}, nil
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
		BasePrice:           input.BasePrice,
		SpecialPrice:        input.SpecialPrice,
		SpecialPriceStartAt: input.SpecialPriceStartAt,
		SpecialPriceEndAt:   input.SpecialPriceEndAt,
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
	if product.MinStock < 0 {
		return ErrInvalidMinStock
	}
	if product.MaxStock != nil && *product.MaxStock < product.MinStock {
		return ErrInvalidMaxStock
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

func (s Service) generateUniqueSKU(ctx context.Context, storeID string) (string, error) {
	const maxAttempts = 20
	for i := 0; i < maxAttempts; i++ {
		candidate := buildEAN13(newBarcode12Digits())
		exists, err := s.repo.SKUExists(ctx, storeID, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", ErrGenerateSKUFailed
}

func (s Service) generateUniqueBarcode(ctx context.Context, storeID string) (string, error) {
	const maxAttempts = 20
	for i := 0; i < maxAttempts; i++ {
		candidate := buildEAN13(newBarcode12Digits())
		exists, err := s.repo.BarcodeExists(ctx, storeID, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", ErrGenerateSKUFailed
}
