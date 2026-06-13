package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/config"
	"pos-backend/internal/modules/warehouse_receipt"
)

func newWarehouseReceiptHandler(cfg config.Config, db *gorm.DB) warehouse_receipt.Handler {
	repo := warehouse_receipt.NewPostgresRepository(db)
	storage := warehouse_receipt.NewMinIOAttachmentStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucketName,
		cfg.MinIOUseSSL,
		cfg.MinIOPublicURL,
	)
	service := warehouse_receipt.NewService(repo, db, storage)
	return warehouse_receipt.NewHandler(service)
}

func registerWarehouseReceiptRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.warehouseReceiptHandler.(warehouse_receipt.Handler)
	protected.Get("/warehouse/receipts", handler.List)
	protected.Post("/warehouse/receipts", handler.Create)
	protected.Get("/warehouse/receipts/:id", handler.GetByID)
	protected.Put("/warehouse/receipts/:id", handler.Update)
	protected.Post("/warehouse/receipts/:id/items", handler.AddItems)
	protected.Put("/warehouse/receipts/:id/items/:item_id", handler.UpdateItem)
	protected.Delete("/warehouse/receipts/:id/items/:item_id", handler.DeleteItem)
	protected.Post("/warehouse/receipts/:id/submit", handler.Submit)
	protected.Post("/warehouse/receipts/:id/reopen", handler.Reopen)
	protected.Post("/warehouse/receipts/:id/confirm", handler.Confirm)
	protected.Post("/warehouse/receipts/:id/cancel", handler.Cancel)
	protected.Post("/warehouse/receipts/:id/attachment", handler.UploadAttachment)
	protected.Get("/warehouse/receipts/:id/stock-impact", handler.StockImpact)
	protected.Get("/warehouse/receipts/:id/print", handler.Print)
	protected.Post("/warehouse/receipts/generate-document-no", handler.GenerateDocumentNo)
}
