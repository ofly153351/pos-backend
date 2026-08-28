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
	g := newStoreGuards(deps)
	protected.Get("/stores/:storeID/stock-count-sessions", g.operate, handler.List)
	protected.Get("/stores/:storeID/stock-count-sessions/:sessionID", g.operate, handler.GetByID)
	protected.Put("/stores/:storeID/stock-count-sessions/:sessionID", g.operate, handler.Save)
	protected.Post("/stores/:storeID/stock-count-sessions/:sessionID/apply", g.operate, handler.Apply)
	protected.Delete("/stores/:storeID/stock-count-sessions/:sessionID", g.operate, handler.Delete)
}
