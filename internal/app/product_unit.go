package app

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/modules/productunit"
)

func newProductUnitHandler(db *sql.DB) productunit.Handler {
	repo := productunit.NewPostgresRepository(db)
	service := productunit.NewService(repo)
	return productunit.NewHandler(service)
}

func registerProductUnitRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.productUnitHandler.(productunit.Handler)
	protected.Post("/stores/:storeID/product-units", handler.Create)
	protected.Get("/stores/:storeID/product-units", handler.List)
	protected.Patch("/stores/:storeID/product-units/:unitID", handler.Update)
	protected.Delete("/stores/:storeID/product-units/:unitID", handler.Delete)
}
