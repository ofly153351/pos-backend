package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/productunit"
)

func newProductUnitHandler(db *gorm.DB) productunit.Handler {
	repo := productunit.NewPostgresRepository(db)
	service := productunit.NewService(repo)
	return productunit.NewHandler(service)
}

func registerProductUnitRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.productUnitHandler.(productunit.Handler)
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/product-units", g.manage, handler.Create)
	protected.Get("/stores/:storeID/product-units", g.manage, handler.List)
	protected.Patch("/stores/:storeID/product-units/:unitID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/product-units/:unitID", g.manage, handler.Delete)
}
