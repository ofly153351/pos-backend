package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/vat"
)

func newVATHandler(db *gorm.DB) vat.Handler {
	repo := vat.NewPostgresRepository(db)
	return vat.NewHandler(vat.NewService(repo))
}

func registerVATRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.vatHandler.(vat.Handler)
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/vat/calculate", g.operate, handler.Calculate)
}
