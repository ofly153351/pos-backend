package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/finance"
)

func newFinanceHandler(db *gorm.DB) finance.Handler {
	repo := finance.NewPostgresRepository(db)
	service := finance.NewService(repo)
	return finance.NewHandler(service)
}

func registerFinanceRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.financeHandler.(finance.Handler)
	g := newStoreGuards(deps)
	protected.Get("/stores/:storeID/finance/pnl", g.operate, handler.GetPnL)
	protected.Get("/stores/:storeID/finance/summary", g.operate, handler.GetSummary)
	protected.Get("/stores/:storeID/finance/inventory", g.operate, handler.GetInventory)
}
