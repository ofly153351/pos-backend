package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/stock"
)

func newStockHandler(db *gorm.DB) stock.Handler {
	repo := stock.NewPostgresRepository(db)
	service := stock.NewService(repo, db)
	return stock.NewHandler(service)
}

func registerStockRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.stockHandler.(stock.Handler)

	protected.Get("/stores/:storeID/stock/products/:productID", handler.GetByProduct)
	protected.Get("/stores/:storeID/stock/locations/:locationID", handler.GetByLocation)
	protected.Get("/stores/:storeID/stock/low-stock", handler.ListLowStock)
}
