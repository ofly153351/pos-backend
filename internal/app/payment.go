package app

import "github.com/gofiber/fiber/v2"

func registerPaymentRoutes(protected fiber.Router, deps appDependencies) {
	g := newStoreGuards(deps)
	protected.Get("/stores/:storeID/promptpay-qr", g.operate, deps.paymentHandler.GenerateQR)
}
