package store

import (
	"mime/multipart"

	pkgstorage "pos-backend/internal/platform/storage"
)

// LogoStorage wraps the platform interface so the rest of the store module
// stays unchanged (SaveStoreLogo / DeleteStoreLogo method names).

type LogoStorage interface {
	SaveStoreLogo(file *multipart.FileHeader) (string, error)
	DeleteStoreLogo(logoURL string) error
}

type logoStorageAdapter struct {
	inner pkgstorage.LogoStorage
}

func (a logoStorageAdapter) SaveStoreLogo(file *multipart.FileHeader) (string, error) {
	return a.inner.SaveLogo(file)
}

func (a logoStorageAdapter) DeleteStoreLogo(logoURL string) error {
	return a.inner.DeleteLogo(logoURL)
}

func NewLocalLogoStorage(baseDir, publicBasePath string) LogoStorage {
	return logoStorageAdapter{inner: pkgstorage.NewLocalLogoStorage(baseDir, publicBasePath, "logos")}
}

func NewMinIOLogoStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicURL string) LogoStorage {
	return logoStorageAdapter{inner: pkgstorage.NewMinIOLogoStorage(endpoint, accessKey, secretKey, bucket, useSSL, publicURL, "logos")}
}
