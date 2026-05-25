package store_test

import (
	"os"
	"path/filepath"
	"testing"

	storemodule "pos-backend/internal/modules/store"
)

func TestLocalLogoStorageDeleteStoreLogoDeletesOnlySafeOwnedLogo(t *testing.T) {
	baseDir := t.TempDir()
	logosDir := filepath.Join(baseDir, "logos")
	if err := os.MkdirAll(logosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ownedPath := filepath.Join(logosDir, "12345678.png")
	if err := os.WriteFile(ownedPath, []byte("owned"), 0o644); err != nil {
		t.Fatal(err)
	}
	defaultPath := filepath.Join(logosDir, "default.png")
	if err := os.WriteFile(defaultPath, []byte("default"), 0o644); err != nil {
		t.Fatal(err)
	}

	storage := storemodule.NewLocalLogoStorage(baseDir, "/uploads")
	if err := storage.DeleteStoreLogo("/uploads/logos/12345678.png"); err != nil {
		t.Fatalf("delete owned logo: %v", err)
	}
	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("expected owned logo removed, stat err=%v", err)
	}
	if err := storage.DeleteStoreLogo("/uploads/logos/default.png"); err != nil {
		t.Fatalf("default logo delete should be ignored, got %v", err)
	}
	if _, err := os.Stat(defaultPath); err != nil {
		t.Fatalf("default logo should remain, stat err=%v", err)
	}
}

func TestLocalLogoStorageDeleteStoreLogoIgnoresTraversalAndExternalPaths(t *testing.T) {
	baseDir := t.TempDir()
	outsidePath := filepath.Join(baseDir, "outside.png")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	storage := storemodule.NewLocalLogoStorage(baseDir, "/uploads")

	for _, candidate := range []string{
		"/uploads/logos/../outside.png",
		"/uploads/../outside.png",
		"https://cdn.example.com/uploads/logos/12345678.png",
		"/other/logos/12345678.png",
	} {
		if err := storage.DeleteStoreLogo(candidate); err != nil {
			t.Fatalf("unsafe delete %q should be ignored, got %v", candidate, err)
		}
	}
	if _, err := os.Stat(outsidePath); err != nil {
		t.Fatalf("outside file should remain, stat err=%v", err)
	}
}
