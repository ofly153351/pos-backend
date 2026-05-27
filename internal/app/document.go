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
	protected.Get("/stores/:storeID/documents/:docID", h.GetDocument)
	protected.Put("/stores/:storeID/documents/:docID/status", h.UpdateDocumentStatus)
	protected.Delete("/stores/:storeID/documents/:docID", h.DeleteDocument)
	protected.Post("/stores/:storeID/documents/bulk", h.BulkAction)
}
