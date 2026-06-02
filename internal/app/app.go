package app

import (
	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/config"
	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/activity_log"
	"pos-backend/internal/modules/auth"
)

type appDependencies struct {
	tokenManager              auth.TokenManager
	authUserRepo              auth.UserRepository
	authHandler               auth.Handler
	storeHandler              any
	productTypeHandler        any
	productUnitHandler        any
	productBrandHandler       any
	productHandler            any
	customerHandler           any
	vatHandler                any
	saleHandler               any
	parkedBillHandler         any
	invoiceHandler            any
	subscriptionHandler       any
	dashboardHandler          any
	warehouseHandler          any
	purchasingHandler         any
	stockMovementHandler      any
	stockHandler              any
	locationHandler           any
	warehouseDashboardHandler any
	warehouseReceiptHandler   any
	receiptSettingsHandler    any
	documentHandler           any
	activityLogHandler        activity_log.Handler
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
	protected.Use(middleware.ActivityLog(deps.activityLogHandler.Service()))
	registerProtectedAuthRoutes(protected, deps)
	registerStoreRoutes(protected, deps)
	registerProductTypeRoutes(protected, deps)
	registerProductUnitRoutes(protected, deps)
	registerProductBrandRoutes(protected, deps)
	registerProductRoutes(protected, deps)
	registerCustomerRoutes(protected, deps)
	registerVATRoutes(protected, deps)
	registerSaleRoutes(protected, deps)
	registerParkedBillRoutes(protected, deps)
	registerInvoiceRoutes(protected, deps)
	registerSubscriptionRoutes(protected, deps)
	registerDashboardRoutes(protected, deps)
	registerWarehouseRoutes(protected, deps)
	registerPurchasingRoutes(protected, deps)
	registerStockMovementRoutes(protected, deps)
	registerStockRoutes(protected, deps)
	registerLocationRoutes(protected, deps)
	registerWarehouseDashboardRoutes(protected, deps)
	registerWarehouseReceiptRoutes(protected, deps)
	registerReceiptSettingsRoutes(protected, deps)
	registerDocumentRoutes(protected, deps)
	registerActivityLogRoutes(protected, deps)

	admin := api.Group("/admin", middleware.AuthRequired(deps.tokenManager, deps.authUserRepo), middleware.RequireRoles(auth.RolePlatformAdmin))
	registerAdminSubscriptionRoutes(admin, deps)
}
