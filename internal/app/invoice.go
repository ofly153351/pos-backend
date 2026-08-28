package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/config"
	"pos-backend/internal/modules/customer"
	"pos-backend/internal/modules/invoice"
)

func newInvoiceHandler(cfg config.Config, db *gorm.DB) invoice.Handler {
	repo := invoice.NewPostgresRepository(db)
	customerRepo := customer.NewPostgresRepository(db)
	storage := invoice.NewMinIOPaymentProofStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucketName,
		cfg.MinIOUseSSL,
		cfg.MinIOPublicURL,
	)
	service := invoice.NewService(repo, customer.NewSaleBenefitResolver(customerRepo), storage)
	return invoice.NewHandler(service)
}

func registerInvoiceRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.invoiceHandler.(invoice.Handler)
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/invoices", g.operate, handler.Create)
	protected.Get("/stores/:storeID/invoices", g.operate, handler.ListByStore)
	protected.Get("/stores/:storeID/invoices/:invoiceID", g.operate, handler.GetByID)
	protected.Post("/stores/:storeID/invoices/:invoiceID/payments", g.operate, handler.AddPayment)
	protected.Get("/stores/:storeID/invoices/:invoiceID/payments/:paymentID/proof", g.operate, handler.ViewPaymentProof)
	protected.Post("/stores/:storeID/invoices/:invoiceID/unpay", g.operate, handler.MarkUnpaid)
	protected.Get("/stores/:storeID/invoices/:invoiceID/pdf", g.operate, handler.ExportPDF)
}
