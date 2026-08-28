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
	g := newStoreGuards(deps)

	protected.Post("/stores/:storeID/locations", g.manage, handler.Create)
	protected.Get("/stores/:storeID/locations", g.access, handler.ListByStore)
	// Static sub-paths must come before /:locationID
	protected.Get("/stores/:storeID/locations/tree", g.access, handler.GetTree)
	protected.Patch("/stores/:storeID/locations/zones", g.manage, handler.RenameZone)
	protected.Delete("/stores/:storeID/locations/zones", g.manage, handler.DeleteZone)
	protected.Patch("/stores/:storeID/locations/floors", g.manage, handler.RenameFloor)
	protected.Delete("/stores/:storeID/locations/floors", g.manage, handler.DeleteFloor)
	protected.Get("/stores/:storeID/locations/:locationID", g.access, handler.GetByID)
	protected.Get("/stores/:storeID/locations/:locationID/products", g.access, handler.ListProducts)
	// Safe-delete lifecycle: read-only assessment that drives the adaptive delete/archive
	// modal. Static "deletion-assessment" suffix, so no collision with GetByID above.
	protected.Get("/stores/:storeID/locations/:locationID/deletion-assessment", g.manage, handler.AssessDeletion)
	protected.Patch("/stores/:storeID/locations/:locationID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/locations/:locationID", g.manage, handler.Delete)
}
