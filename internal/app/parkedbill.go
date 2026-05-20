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
	protected.Post("/stores/:storeID/parked-bills", handler.Create)
	protected.Get("/stores/:storeID/parked-bills", handler.List)
	protected.Get("/stores/:storeID/parked-bills/:parkedBillID", handler.GetByID)
	protected.Delete("/stores/:storeID/parked-bills/:parkedBillID", handler.Delete)
}
