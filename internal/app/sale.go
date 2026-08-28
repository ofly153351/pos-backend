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
	settingsRepo := sale.NewDBSettingsRepo(db)
	service := sale.NewService(saleRepo, customer.NewSaleBenefitResolver(customerRepo), settingsRepo)
	return sale.NewHandler(service)
}

func registerSaleRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.saleHandler.(sale.Handler)
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/sales", g.operate, handler.Create)
	protected.Get("/stores/:storeID/sales", g.operate, handler.ListByStore)
	protected.Get("/stores/:storeID/sales/:saleID", g.operate, handler.GetByID)
	protected.Get("/stores/:storeID/sales/:saleID/receipt", g.operate, handler.Receipt)
	protected.Get("/stores/:storeID/sales/:saleID/receipt/preview", g.operate, handler.ReceiptPreview)
	protected.Get("/stores/:storeID/sales/:saleID/document", g.operate, handler.Document)
	protected.Post("/stores/:storeID/sales/:saleID/void", g.manage, handler.VoidSale)
	protected.Post("/stores/:storeID/sales/:saleID/returns", g.manage, handler.CreateReturn)
}
