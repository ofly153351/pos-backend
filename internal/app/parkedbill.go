package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/parkedbill"
)

func newParkedBillHandler(db *gorm.DB) parkedbill.Handler {
	repo := parkedbill.NewPostgresRepository(db)
	service := parkedbill.NewService(repo)
	return parkedbill.NewHandler(service)
}

func registerParkedBillRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.parkedBillHandler.(parkedbill.Handler)
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/parked-bills", g.operate, handler.Create)
	protected.Get("/stores/:storeID/parked-bills", g.operate, handler.List)
	protected.Get("/stores/:storeID/parked-bills/:parkedBillID", g.operate, handler.GetByID)
	protected.Delete("/stores/:storeID/parked-bills/:parkedBillID", g.operate, handler.Delete)
}
