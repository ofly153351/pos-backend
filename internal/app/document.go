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
	g := newStoreGuards(deps)

	protected.Get("/stores/:storeID/documents", g.operate, h.ListDocuments)
	protected.Post("/stores/:storeID/documents", g.operate, h.CreateDocument)
	// Issue a persisted document (e.g. TAX_INVOICE) from a POS sale, copying the
	// sale's authoritative totals so the document matches the receipt exactly.
	protected.Post("/stores/:storeID/sales/:saleID/documents", g.operate, h.CreateFromSale)
	protected.Get("/stores/:storeID/documents/statement", g.operate, h.GetStatementPDF)
	protected.Get("/stores/:storeID/documents/:docID", g.operate, h.GetDocument)
	protected.Get("/stores/:storeID/documents/:docID/print", g.operate, h.PrintDocument)
	protected.Get("/stores/:storeID/documents/:docID/related", g.operate, h.RelatedDocuments)
	protected.Get("/stores/:storeID/documents/:docID/wht-cert", g.operate, h.PrintWHTCert)
	protected.Get("/stores/:storeID/documents/:docID/pdf", g.operate, h.GetDocumentPDF)
	protected.Put("/stores/:storeID/documents/:docID/status", g.operate, h.UpdateDocumentStatus)
	protected.Put("/stores/:storeID/documents/:docID/payment-status", g.operate, h.UpdateDocumentPaymentStatus)
	protected.Delete("/stores/:storeID/documents/:docID", g.operate, h.DeleteDocument)
	protected.Post("/stores/:storeID/documents/bulk", g.operate, h.BulkAction)
	protected.Post("/stores/:storeID/documents/:docID/pay", g.operate, h.PayInvoice)
	protected.Post("/stores/:storeID/documents/:docID/convert-tax", g.operate, h.ConvertToTaxInvoice)
	protected.Post("/stores/:storeID/documents/:docID/convert-do", g.operate, h.ConvertToDeliveryOrder)
	protected.Post("/stores/:storeID/documents/:docID/convert", g.operate, h.ConvertQuotation)
	protected.Post("/stores/:storeID/documents/:docID/convert-to", g.operate, h.Convert)
}
