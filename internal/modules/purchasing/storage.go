package purchasing

import (
	"mime/multipart"

	pkgstorage "pos-backend/internal/platform/storage"
)

type SupplierLogoStorage interface {
	SaveSupplierLogo(file *multipart.FileHeader) (string, error)
	DeleteSupplierLogo(logoURL string) error
}

type supplierLogoAdapter struct {
	inner pkgstorage.LogoStorage
}

func (a supplierLogoAdapter) SaveSupplierLogo(file *multipart.FileHeader) (string, error) {
	return a.inner.SaveLogo(file)
}

func (a supplierLogoAdapter) DeleteSupplierLogo(logoURL string) error {
	return a.inner.DeleteLogo(logoURL)
}

func NewLocalSupplierLogoStorage(baseDir, publicBasePath string) SupplierLogoStorage {
	return supplierLogoAdapter{inner: pkgstorage.NewLocalLogoStorage(baseDir, publicBasePath, "supplier-logos")}
}

func NewMinIOSupplierLogoStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicURL string) SupplierLogoStorage {
	return supplierLogoAdapter{inner: pkgstorage.NewMinIOLogoStorage(endpoint, accessKey, secretKey, bucket, useSSL, publicURL, "supplier-logos")}
}
