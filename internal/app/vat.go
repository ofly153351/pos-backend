package app

import (
	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/modules/vat"
)

func newVATHandler() vat.Handler {
	return vat.NewHandler(vat.NewService())
}

func registerVATRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.vatHandler.(vat.Handler)
	protected.Post("/stores/:storeID/vat/calculate", handler.Calculate)
}
