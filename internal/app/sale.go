package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/sale"
)

func newSaleHandler(db *gorm.DB) sale.Handler {
	repo := sale.NewPostgresRepository(db)
	service := sale.NewService(repo)
	return sale.NewHandler(service)
}

func registerSaleRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.saleHandler.(sale.Handler)
	protected.Post("/stores/:storeID/sales", handler.Create)
	protected.Get("/stores/:storeID/sales", handler.ListByStore)
	protected.Get("/stores/:storeID/sales/:saleID", handler.GetByID)
}
