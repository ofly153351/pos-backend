package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/stockcount"
)

func newStockCountHandler(db *gorm.DB) stockcount.Handler {
	repo := stockcount.NewPostgresRepository(db)
	service := stockcount.NewService(repo, db)
	return stockcount.NewHandler(service)
}

func registerStockCountRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.stockCountHandler.(stockcount.Handler)
	protected.Get("/stores/:storeID/stock-count-sessions", handler.List)
	protected.Get("/stores/:storeID/stock-count-sessions/:sessionID", handler.GetByID)
	protected.Put("/stores/:storeID/stock-count-sessions/:sessionID", handler.Save)
	protected.Post("/stores/:storeID/stock-count-sessions/:sessionID/apply", handler.Apply)
	protected.Delete("/stores/:storeID/stock-count-sessions/:sessionID", handler.Delete)
}
