package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/document"
)

func newDocumentHandler(db *gorm.DB) document.Handler {
	repo := document.NewRepository(db)
	service := document.NewService(repo, db)
	return document.NewHandler(service)
}

func registerDocumentRoutes(protected fiber.Router, deps appDependencies) {
	h := deps.documentHandler.(document.Handler)

	protected.Get("/stores/:storeID/documents", h.ListDocuments)
	protected.Post("/stores/:storeID/documents", h.CreateDocument)
	// Issue a persisted document (e.g. TAX_INVOICE) from a POS sale, copying the
	// sale's authoritative totals so the document matches the receipt exactly.
	protected.Post("/stores/:storeID/sales/:saleID/documents", h.CreateFromSale)
	protected.Get("/stores/:storeID/documents/statement", h.GetStatementPDF)
	protected.Get("/stores/:storeID/documents/:docID", h.GetDocument)
	protected.Get("/stores/:storeID/documents/:docID/print", h.PrintDocument)
	protected.Get("/stores/:storeID/documents/:docID/related", h.RelatedDocuments)
	protected.Get("/stores/:storeID/documents/:docID/wht-cert", h.PrintWHTCert)
	protected.Get("/stores/:storeID/documents/:docID/pdf", h.GetDocumentPDF)
	protected.Put("/stores/:storeID/documents/:docID/status", h.UpdateDocumentStatus)
	protected.Delete("/stores/:storeID/documents/:docID", h.DeleteDocument)
	protected.Post("/stores/:storeID/documents/bulk", h.BulkAction)
	protected.Post("/stores/:storeID/documents/:docID/pay", h.PayInvoice)
	protected.Post("/stores/:storeID/documents/:docID/convert-tax", h.ConvertToTaxInvoice)
	protected.Post("/stores/:storeID/documents/:docID/convert-do", h.ConvertToDeliveryOrder)
	protected.Post("/stores/:storeID/documents/:docID/convert", h.ConvertQuotation)
	protected.Post("/stores/:storeID/documents/:docID/convert-to", h.Convert)
}
