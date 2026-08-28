package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/stock_movement"
)

func newStockMovementHandler(db *gorm.DB) stock_movement.Handler {
	repo := stock_movement.NewPostgresRepository(db)
	service := stock_movement.NewService(repo, db)
	return stock_movement.NewHandler(service)
}

func registerStockMovementRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.stockMovementHandler.(stock_movement.Handler)
	g := newStoreGuards(deps)

	protected.Post("/stores/:storeID/stock-movements/in", g.manage, handler.AddStock)
	protected.Post("/stores/:storeID/stock-movements/out", g.manage, handler.RemoveStock)
	protected.Post("/stores/:storeID/stock-movements/transfer", g.manage, handler.TransferStock)
	protected.Post("/stores/:storeID/stock-movements/adjust", g.manage, handler.AdjustStock)
	protected.Get("/stores/:storeID/stock-movements", g.manage, handler.ListMovements)
}
