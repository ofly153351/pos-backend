package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/config"
	"pos-backend/internal/modules/product"
)

func newProductHandler(cfg config.Config, db *gorm.DB) product.Handler {
	repo := product.NewPostgresRepository(db)
	storage := product.NewMinIOImageStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucketName,
		cfg.MinIOUseSSL,
		cfg.MinIOPublicURL,
	)
	service := product.NewService(repo, storage, db)
	return product.NewHandler(service)
}

func registerProductRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.productHandler.(product.Handler)
	protected.Post("/stores/:storeID/products", handler.Create)
	protected.Post("/stores/:storeID/products/generate-missing-barcodes", handler.GenerateMissingBarcodes)
	protected.Get("/stores/:storeID/products", handler.ListByStore)
	protected.Get("/stores/:storeID/products/:productID", handler.GetByID)
	protected.Patch("/stores/:storeID/products/:productID", handler.Update)
	protected.Delete("/stores/:storeID/products/:productID", handler.Delete)
}
