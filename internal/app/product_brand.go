package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/productbrand"
)

func newProductBrandHandler(db *gorm.DB) productbrand.Handler {
	repo := productbrand.NewPostgresRepository(db)
	service := productbrand.NewService(repo)
	return productbrand.NewHandler(service)
}

func registerProductBrandRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.productBrandHandler.(productbrand.Handler)
	protected.Post("/stores/:storeID/product-brands", handler.Create)
	protected.Get("/stores/:storeID/product-brands", handler.ListByStore)
	protected.Patch("/stores/:storeID/product-brands/:brandID", handler.Update)
	protected.Delete("/stores/:storeID/product-brands/:brandID", handler.Delete)
}
