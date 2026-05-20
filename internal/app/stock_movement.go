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

	protected.Post("/stores/:storeID/stock-movements/in", handler.AddStock)
	protected.Post("/stores/:storeID/stock-movements/out", handler.RemoveStock)
	protected.Post("/stores/:storeID/stock-movements/transfer", handler.TransferStock)
	protected.Post("/stores/:storeID/stock-movements/adjust", handler.AdjustStock)
	protected.Get("/stores/:storeID/stock-movements", handler.ListMovements)
}
