package product_test

import (
	"os"
	"path/filepath"
	"testing"

	productmodule "pos-backend/internal/modules/product"
)

func TestLocalImageStorageDeleteProductImageDeletesOnlySafeOwnedImage(t *testing.T) {
	baseDir := t.TempDir()
	productsDir := filepath.Join(baseDir, "products")
	if err := os.MkdirAll(productsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ownedPath := filepath.Join(productsDir, "pd-12345678.png")
	if err := os.WriteFile(ownedPath, []byte("owned"), 0o644); err != nil {
		t.Fatal(err)
	}
	defaultPath := filepath.Join(productsDir, "default.png")
	if err := os.WriteFile(defaultPath, []byte("default"), 0o644); err != nil {
		t.Fatal(err)
	}

	storage := productmodule.NewLocalImageStorage(baseDir, "/uploads")
	if err := storage.DeleteProductImage("/uploads/products/pd-12345678.png"); err != nil {
		t.Fatalf("delete owned product image: %v", err)
	}
	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("expected owned product image removed, stat err=%v", err)
	}
	if err := storage.DeleteProductImage("/uploads/products/default.png"); err != nil {
		t.Fatalf("default product image delete should be ignored, got %v", err)
	}
	if _, err := os.Stat(defaultPath); err != nil {
		t.Fatalf("default product image should remain, stat err=%v", err)
	}
}

func TestLocalImageStorageDeleteProductImageIgnoresTraversalAndExternalPaths(t *testing.T) {
	baseDir := t.TempDir()
	outsidePath := filepath.Join(baseDir, "outside.png")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	storage := productmodule.NewLocalImageStorage(baseDir, "/uploads")

	for _, candidate := range []string{
		"/uploads/products/../outside.png",
		"/uploads/../outside.png",
		"https://cdn.example.com/uploads/products/pd-12345678.png",
		"/other/products/pd-12345678.png",
	} {
		if err := storage.DeleteProductImage(candidate); err != nil {
			t.Fatalf("unsafe delete %q should be ignored, got %v", candidate, err)
		}
	}
	if _, err := os.Stat(outsidePath); err != nil {
		t.Fatalf("outside file should remain, stat err=%v", err)
	}
}
