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
	protected.Post("/stores/:storeID/product-types", handler.Create)
	protected.Get("/stores/:storeID/product-types", handler.ListByStore)
	protected.Patch("/stores/:storeID/product-types/:productTypeID", handler.Update)
	protected.Delete("/stores/:storeID/product-types/:productTypeID", handler.Delete)
}
