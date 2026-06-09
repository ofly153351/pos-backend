package app

import "github.com/gofiber/fiber/v2"

func registerPaymentRoutes(protected fiber.Router, deps appDependencies) {
	protected.Get("/stores/:storeID/promptpay-qr", deps.paymentHandler.GenerateQR)
}
