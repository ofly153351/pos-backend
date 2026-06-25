package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/creditsale"
	"pos-backend/internal/modules/customer"
	"pos-backend/internal/modules/sale"
)

func newCreditSaleHandler(db *gorm.DB) creditsale.Handler {
	// Reuse the real sale pipeline (server-authoritative pricing, customer benefit
	// resolution, stock deduction) so a credit sale is a genuine product-backed sale.
	saleRepo := sale.NewPostgresRepository(db)
	customerRepo := customer.NewPostgresRepository(db)
	settingsRepo := sale.NewDBSettingsRepo(db)
	saleService := sale.NewService(saleRepo, customer.NewSaleBenefitResolver(customerRepo), settingsRepo)

	repo := creditsale.NewPostgresRepository(db)
	service := creditsale.NewService(repo, saleService)
	return creditsale.NewHandler(service)
}

func registerCreditSaleRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.creditSaleHandler.(creditsale.Handler)
	// /summary and /aging before /:creditSaleID so Fiber matches them first.
	protected.Get("/stores/:storeID/credit-sales/summary", handler.Summary)
	protected.Get("/stores/:storeID/credit-sales/aging", handler.Aging)
	protected.Get("/stores/:storeID/credit-sales", handler.List)
	protected.Post("/stores/:storeID/credit-sales", handler.Create)
	protected.Get("/stores/:storeID/credit-sales/:creditSaleID", handler.GetByID)
	protected.Post("/stores/:storeID/credit-sales/:creditSaleID/payments", handler.AddPayment)
	protected.Post("/stores/:storeID/credit-sales/:creditSaleID/cancel", handler.Cancel)
	protected.Get("/stores/:storeID/credit-sales/:creditSaleID/statement", handler.Statement)
	protected.Get("/stores/:storeID/credit-sales/:creditSaleID/bill", handler.Bill)
}
