package product_test

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"pos-backend/internal/modules/auth"
	productmodule "pos-backend/internal/modules/product"
)

type fakeProductRepo struct {
	product   productmodule.Product
	updateErr error
}

func (r *fakeProductRepo) Create(ctx context.Context, product productmodule.Product) (productmodule.Product, error) {
	r.product = product
	return product, nil
}
func (r *fakeProductRepo) ListByStore(ctx context.Context, storeID string, page, limit int, stockStatus string) ([]productmodule.Product, int64, error) {
	return []productmodule.Product{r.product}, 1, nil
}
func (r *fakeProductRepo) GetByID(ctx context.Context, storeID, productID string) (productmodule.Product, error) {
	return r.product, nil
}
func (r *fakeProductRepo) Update(ctx context.Context, product productmodule.Product) (productmodule.Product, error) {
	if r.updateErr != nil {
		return productmodule.Product{}, r.updateErr
	}
	r.product = product
	return product, nil
}
func (r *fakeProductRepo) UpdateSKU(ctx context.Context, storeID, productID, sku string, updatedAt time.Time) error {
	return nil
}
func (r *fakeProductRepo) SKUExists(ctx context.Context, storeID, sku string) (bool, error) {
	return false, nil
}
func (r *fakeProductRepo) BarcodeExists(ctx context.Context, storeID, barcode string) (bool, error) {
	return false, nil
}
func (r *fakeProductRepo) ListProductIDsWithoutSKU(ctx context.Context, storeID string) ([]string, error) {
	return nil, nil
}
func (r *fakeProductRepo) Delete(ctx context.Context, storeID, productID string) error { return nil }
func (r *fakeProductRepo) SoftDelete(ctx context.Context, storeID, productID string) error {
	return nil
}
func (r *fakeProductRepo) ExistsByID(ctx context.Context, storeID, productID string) (bool, error) {
	return true, nil
}
func (r *fakeProductRepo) ProductTypeExists(ctx context.Context, storeID, productTypeID string) (bool, error) {
	return true, nil
}
func (r *fakeProductRepo) ProductUnitExists(ctx context.Context, storeID, productUnitID string) (bool, error) {
	return true, nil
}
func (r *fakeProductRepo) BrandExists(ctx context.Context, storeID, brandID string) (bool, error) {
	return true, nil
}
func (r *fakeProductRepo) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	return true, nil
}

type fakeImageStorage struct {
	saveURL   string
	deleted   []string
	deleteErr error
}

func (s *fakeImageStorage) SaveProductImage(file *multipart.FileHeader) (string, error) {
	return s.saveURL, nil
}
func (s *fakeImageStorage) DeleteProductImage(imageURL string) error {
	s.deleted = append(s.deleted, imageURL)
	return s.deleteErr
}

func baseProduct() productmodule.Product {
	return productmodule.Product{ID: "product-1", StoreID: "store-1", Name: "Coffee", ProductUnitID: "unit-1", BrandID: "brand-1", BasePrice: 10, ImageURL: "/uploads/products/old.png", IsActive: true}
}

func TestUpdateProductImageDeletesOldImageAfterSuccessfulDBUpdate(t *testing.T) {
	repo := &fakeProductRepo{product: baseProduct()}
	storage := &fakeImageStorage{saveURL: "/uploads/products/new.png"}
	svc := productmodule.NewService(repo, storage)

	updated, err := svc.Update(context.Background(), auth.Claims{UserID: "user-1", Role: "owner"}, "store-1", "product-1", productmodule.UpdateProductRequest{ImageFile: &multipart.FileHeader{Filename: "new.png"}})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.ImageURL != "/uploads/products/new.png" {
		t.Fatalf("expected updated image URL, got %q", updated.ImageURL)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "/uploads/products/old.png" {
		t.Fatalf("expected old image delete, got %#v", storage.deleted)
	}
}

func TestUpdateProductImageDeletesNewImageWhenDBUpdateFails(t *testing.T) {
	dbErr := errors.New("db failed")
	repo := &fakeProductRepo{product: baseProduct(), updateErr: dbErr}
	storage := &fakeImageStorage{saveURL: "/uploads/products/new.png"}
	svc := productmodule.NewService(repo, storage)

	_, err := svc.Update(context.Background(), auth.Claims{UserID: "user-1", Role: "owner"}, "store-1", "product-1", productmodule.UpdateProductRequest{ImageFile: &multipart.FileHeader{Filename: "new.png"}})
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "/uploads/products/new.png" {
		t.Fatalf("expected compensating delete of new image, got %#v", storage.deleted)
	}
}

func TestUpdateProductImageOldDeleteFailureDoesNotBlockResponse(t *testing.T) {
	repo := &fakeProductRepo{product: baseProduct()}
	storage := &fakeImageStorage{saveURL: "/uploads/products/new.png", deleteErr: errors.New("delete failed")}
	svc := productmodule.NewService(repo, storage)

	updated, err := svc.Update(context.Background(), auth.Claims{UserID: "user-1", Role: "owner"}, "store-1", "product-1", productmodule.UpdateProductRequest{ImageFile: &multipart.FileHeader{Filename: "new.png"}})
	if err != nil {
		t.Fatalf("old delete failure should not block update, got %v", err)
	}
	if updated.ImageURL != "/uploads/products/new.png" {
		t.Fatalf("expected updated image URL, got %q", updated.ImageURL)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "/uploads/products/old.png" {
		t.Fatalf("expected old image delete attempt, got %#v", storage.deleted)
	}
}
