package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/warehouse_dashboard"
)

func newWarehouseDashboardHandler(db *gorm.DB) warehouse_dashboard.Handler {
	repo := warehouse_dashboard.NewPostgresRepository(db)
	service := warehouse_dashboard.NewService(repo)
	return warehouse_dashboard.NewHandler(service)
}

func registerWarehouseDashboardRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.warehouseDashboardHandler.(warehouse_dashboard.Handler)
	g := newStoreGuards(deps)
	protected.Get("/stores/:storeID/dashboard/warehouse", g.operate, handler.GetDashboard)
}
