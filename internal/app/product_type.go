package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/producttype"
)

func newProductTypeHandler(db *gorm.DB) producttype.Handler {
	repo := producttype.NewPostgresRepository(db)
	service := producttype.NewService(repo)
	return producttype.NewHandler(service)
}

func registerProductTypeRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.productTypeHandler.(producttype.Handler)
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/product-types", g.manage, handler.Create)
	protected.Get("/stores/:storeID/product-types", g.manage, handler.ListByStore)
	protected.Patch("/stores/:storeID/product-types/:productTypeID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/product-types/:productTypeID", g.manage, handler.Delete)
}
