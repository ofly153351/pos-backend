package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/dashboard"
)

func newDashboardHandler(db *gorm.DB) dashboard.Handler {
	repo := dashboard.NewPostgresRepository(db)
	service := dashboard.NewService(repo)
	return dashboard.NewHandler(service)
}

func registerDashboardRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.dashboardHandler.(dashboard.Handler)
	protected.Get("/stores/:storeID/dashboard", handler.GetOverview)
}
