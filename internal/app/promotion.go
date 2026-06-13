package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/promotion"
)

func newPromotionHandler(db *gorm.DB) promotion.Handler {
	repo := promotion.NewPostgresRepository(db)
	service := promotion.NewService(repo)
	return promotion.NewHandler(service)
}

func registerPromotionRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.promotionHandler.(promotion.Handler)
	protected.Get("/stores/:storeID/promotions", handler.List)
	protected.Post("/stores/:storeID/promotions", handler.Create)
	protected.Put("/stores/:storeID/promotions/:promotionID", handler.Update)
	protected.Delete("/stores/:storeID/promotions/:promotionID", handler.Delete)
}
