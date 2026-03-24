package app

import (
	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/config"
	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/auth"
)

type appDependencies struct {
	tokenManager        auth.TokenManager
	authUserRepo        auth.UserRepository
	authHandler         auth.Handler
	storeHandler        any
	productTypeHandler  any
	productUnitHandler  any
	productHandler      any
	customerHandler     any
	vatHandler          any
	saleHandler         any
	invoiceHandler      any
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
	registerVersionedAPIRoutes(app.Group("/api/v1"), deps)
	registerVersionedAPIRoutes(app.Group("/api"), deps)
}

func registerVersionedAPIRoutes(api fiber.Router, deps appDependencies) {
	registerAuthRoutes(api, deps)

	protected := api.Group("", middleware.AuthRequired(deps.tokenManager, deps.authUserRepo))
	registerProtectedAuthRoutes(protected, deps)
	registerStoreRoutes(protected, deps)
	registerProductTypeRoutes(protected, deps)
	registerProductUnitRoutes(protected, deps)
	registerProductRoutes(protected, deps)
	registerCustomerRoutes(protected, deps)
	registerVATRoutes(protected, deps)
	registerSaleRoutes(protected, deps)
	registerInvoiceRoutes(protected, deps)
	registerSubscriptionRoutes(protected, deps)

	admin := api.Group("/admin", middleware.AuthRequired(deps.tokenManager, deps.authUserRepo), middleware.RequireRoles(auth.RolePlatformAdmin))
	registerAdminSubscriptionRoutes(admin, deps)
}
