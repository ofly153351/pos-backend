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
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/product-brands", g.manage, handler.Create)
	protected.Get("/stores/:storeID/product-brands", g.manage, handler.ListByStore)
	protected.Patch("/stores/:storeID/product-brands/:brandID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/product-brands/:brandID", g.manage, handler.Delete)
}
