package app

import (
	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/config"
	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/auth"
)

type appDependencies struct {
	tokenManager        auth.TokenManager
	authHandler         auth.Handler
	storeHandler        any
	productTypeHandler  any
	productUnitHandler  any
	productHandler      any
	saleHandler         any
	subscriptionHandler any
}

func registerBaseRoutes(app *fiber.App, cfg config.Config) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})
}

func registerAPIRoutes(app *fiber.App, deps appDependencies) {
	api := app.Group("/api/v1")

	registerAuthRoutes(api, deps)

	protected := api.Group("", middleware.AuthRequired(deps.tokenManager))
	registerStoreRoutes(protected, deps)
	registerProductTypeRoutes(protected, deps)
	registerProductUnitRoutes(protected, deps)
	registerProductRoutes(protected, deps)
	registerSaleRoutes(protected, deps)
	registerSubscriptionRoutes(protected, deps)

	admin := api.Group("/admin", middleware.AuthRequired(deps.tokenManager), middleware.RequireRoles(auth.RolePlatformAdmin))
	registerAdminSubscriptionRoutes(admin, deps)
}
