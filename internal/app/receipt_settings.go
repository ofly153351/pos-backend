package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	receipt_settings "pos-backend/internal/modules/receipt_settings"
	"pos-backend/internal/modules/store"
)

func newReceiptSettingsHandler(db *gorm.DB) receipt_settings.Handler {
	repo := receipt_settings.NewPostgresRepository(db)
	storeRepo := store.NewPostgresRepository(db)
	svc := receipt_settings.NewService(repo, storeRepo)
	return receipt_settings.NewHandler(svc)
}

func registerReceiptSettingsRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.receiptSettingsHandler.(receipt_settings.Handler)
	g := newStoreGuards(deps)
	protected.Get("/stores/:storeID/receipt-settings", g.manage, handler.Get)
	protected.Put("/stores/:storeID/receipt-settings", g.manage, handler.Update)
	protected.Post("/stores/:storeID/receipt-settings/preview", g.manage, handler.PreviewHTML)
}
