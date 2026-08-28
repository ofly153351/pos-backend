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
	g := newStoreGuards(deps)
	protected.Get("/stores/:storeID/promotions", g.operate, handler.List)
	protected.Post("/stores/:storeID/promotions", g.manage, handler.Create)
	protected.Put("/stores/:storeID/promotions/:promotionID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/promotions/:promotionID", g.manage, handler.Delete)
}
