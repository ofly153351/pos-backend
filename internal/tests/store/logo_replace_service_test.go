package store_test

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"pos-backend/internal/modules/auth"
	storemodule "pos-backend/internal/modules/store"
)

type fakeStoreRepo struct {
	store     storemodule.Store
	updateErr error
}

func (r *fakeStoreRepo) CreateWithOwner(ctx context.Context, store storemodule.Store, ownerUserID string, planCode string) (storemodule.Store, error) {
	r.store = store
	return store, nil
}

func (r *fakeStoreRepo) GetByID(ctx context.Context, storeID string) (storemodule.Store, error) {
	return r.store, nil
}

func (r *fakeStoreRepo) ListByUser(ctx context.Context, userID, role string) ([]storemodule.Store, error) {
	return []storemodule.Store{r.store}, nil
}

func (r *fakeStoreRepo) Update(ctx context.Context, storeID string, update storemodule.Store) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.store = update
	return nil
}

func (r *fakeStoreRepo) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	return true, nil
}

type fakeLogoStorage struct {
	saveURL   string
	deleted   []string
	deleteErr error
}

func (s *fakeLogoStorage) SaveStoreLogo(file *multipart.FileHeader) (string, error) {
	return s.saveURL, nil
}

func (s *fakeLogoStorage) DeleteStoreLogo(logoURL string) error {
	s.deleted = append(s.deleted, logoURL)
	return s.deleteErr
}

func TestUpdateStoreLogoDeletesOldLogoAfterSuccessfulDBUpdate(t *testing.T) {
	repo := &fakeStoreRepo{store: storemodule.Store{ID: "store-1", Name: "Old", CurrencyCode: "THB", LogoURL: "/uploads/logos/old.png", CreatedAt: time.Now()}}
	storage := &fakeLogoStorage{saveURL: "/uploads/logos/new.png"}
	svc := storemodule.NewService(repo, storage)
	name := "New"

	updated, err := svc.Update(context.Background(), auth.Claims{UserID: "user-1", Role: "owner"}, "store-1", storemodule.UpdateStoreRequest{Name: &name, LogoFile: &multipart.FileHeader{Filename: "new.png"}})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.LogoURL != "/uploads/logos/new.png" {
		t.Fatalf("expected updated logo URL, got %q", updated.LogoURL)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "/uploads/logos/old.png" {
		t.Fatalf("expected old logo delete, got %#v", storage.deleted)
	}
}

func TestUpdateStoreLogoDeletesNewLogoWhenDBUpdateFails(t *testing.T) {
	dbErr := errors.New("db failed")
	repo := &fakeStoreRepo{store: storemodule.Store{ID: "store-1", Name: "Old", CurrencyCode: "THB", LogoURL: "/uploads/logos/old.png"}, updateErr: dbErr}
	storage := &fakeLogoStorage{saveURL: "/uploads/logos/new.png"}
	svc := storemodule.NewService(repo, storage)

	_, err := svc.Update(context.Background(), auth.Claims{UserID: "user-1", Role: "owner"}, "store-1", storemodule.UpdateStoreRequest{LogoFile: &multipart.FileHeader{Filename: "new.png"}})
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "/uploads/logos/new.png" {
		t.Fatalf("expected compensating delete of new logo, got %#v", storage.deleted)
	}
}

func TestUpdateStoreLogoOldDeleteFailureDoesNotBlockResponse(t *testing.T) {
	repo := &fakeStoreRepo{store: storemodule.Store{ID: "store-1", Name: "Old", CurrencyCode: "THB", LogoURL: "/uploads/logos/old.png"}}
	storage := &fakeLogoStorage{saveURL: "/uploads/logos/new.png", deleteErr: errors.New("delete failed")}
	svc := storemodule.NewService(repo, storage)

	updated, err := svc.Update(context.Background(), auth.Claims{UserID: "user-1", Role: "owner"}, "store-1", storemodule.UpdateStoreRequest{LogoFile: &multipart.FileHeader{Filename: "new.png"}})
	if err != nil {
		t.Fatalf("old delete failure should not block update, got %v", err)
	}
	if updated.LogoURL != "/uploads/logos/new.png" {
		t.Fatalf("expected updated logo URL, got %q", updated.LogoURL)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "/uploads/logos/old.png" {
		t.Fatalf("expected old logo delete attempt, got %#v", storage.deleted)
	}
}
