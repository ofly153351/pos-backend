package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/location"
)

func newLocationHandler(db *gorm.DB) location.Handler {
	repo := location.NewPostgresRepository(db)
	service := location.NewService(repo, db)
	return location.NewHandler(service)
}

func registerLocationRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.locationHandler.(location.Handler)

	protected.Post("/stores/:storeID/locations", handler.Create)
	protected.Get("/stores/:storeID/locations", handler.ListByStore)
	protected.Get("/stores/:storeID/locations/:locationID", handler.GetByID)
	protected.Patch("/stores/:storeID/locations/:locationID", handler.Update)
	protected.Delete("/stores/:storeID/locations/:locationID", handler.Delete)
}
