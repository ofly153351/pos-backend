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
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/products", g.manage, handler.Create)
	protected.Post("/stores/:storeID/products/generate-missing-barcodes", g.manage, handler.GenerateMissingBarcodes)
	protected.Get("/stores/:storeID/products", g.operate, handler.ListByStore)
	protected.Get("/stores/:storeID/products/:productID", g.manage, handler.GetByID)
	protected.Patch("/stores/:storeID/products/:productID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/products/:productID", g.manage, handler.Delete)
}
