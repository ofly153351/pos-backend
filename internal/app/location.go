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
	// Static sub-paths must come before /:locationID
	protected.Get("/stores/:storeID/locations/tree", handler.GetTree)
	protected.Patch("/stores/:storeID/locations/zones", handler.RenameZone)
	protected.Delete("/stores/:storeID/locations/zones", handler.DeleteZone)
	protected.Patch("/stores/:storeID/locations/floors", handler.RenameFloor)
	protected.Delete("/stores/:storeID/locations/floors", handler.DeleteFloor)
	protected.Get("/stores/:storeID/locations/:locationID", handler.GetByID)
	protected.Get("/stores/:storeID/locations/:locationID/products", handler.ListProducts)
	protected.Patch("/stores/:storeID/locations/:locationID", handler.Update)
	protected.Delete("/stores/:storeID/locations/:locationID", handler.Delete)
}
