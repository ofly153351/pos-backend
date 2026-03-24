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
	protected.Post("/stores/:storeID/invoices", handler.Create)
	protected.Get("/stores/:storeID/invoices", handler.ListByStore)
	protected.Get("/stores/:storeID/invoices/:invoiceID", handler.GetByID)
	protected.Post("/stores/:storeID/invoices/:invoiceID/payments", handler.AddPayment)
	protected.Get("/stores/:storeID/invoices/:invoiceID/payments/:paymentID/proof", handler.ViewPaymentProof)
	protected.Post("/stores/:storeID/invoices/:invoiceID/unpay", handler.MarkUnpaid)
	protected.Get("/stores/:storeID/invoices/:invoiceID/pdf", handler.ExportPDF)
}
