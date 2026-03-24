package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/customer"
	"pos-backend/internal/modules/sale"
)

func newSaleHandler(db *gorm.DB) sale.Handler {
	saleRepo := sale.NewPostgresRepository(db)
	customerRepo := customer.NewPostgresRepository(db)
	service := sale.NewService(saleRepo, customer.NewSaleBenefitResolver(customerRepo))
	return sale.NewHandler(service)
}

func registerSaleRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.saleHandler.(sale.Handler)
	protected.Post("/stores/:storeID/sales", handler.Create)
	protected.Get("/stores/:storeID/sales", handler.ListByStore)
	protected.Get("/stores/:storeID/sales/:saleID", handler.GetByID)
	protected.Get("/stores/:storeID/sales/:saleID/receipt", handler.Receipt)
	protected.Get("/stores/:storeID/sales/:saleID/receipt/preview", handler.ReceiptPreview)
}
