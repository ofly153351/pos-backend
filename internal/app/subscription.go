package app

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/modules/subscription"
)

func newSubscriptionHandler(db *sql.DB) subscription.Handler {
	repo := subscription.NewPostgresRepository(db)
	service := subscription.NewService(repo)
	return subscription.NewHandler(service)
}

func registerSubscriptionRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.subscriptionHandler.(subscription.Handler)
	protected.Get("/subscriptions/plans", handler.ListPlans)
	protected.Get("/stores/:storeID/subscription", handler.GetCurrentByStore)
	protected.Put("/stores/:storeID/subscription", handler.ChangePlan)
}

func registerAdminSubscriptionRoutes(admin fiber.Router, deps appDependencies) {
	handler := deps.subscriptionHandler.(subscription.Handler)
	admin.Get("/subscriptions", handler.AdminListAll)
	admin.Get("/stores/:storeID/subscription", handler.GetCurrentByStore)
	admin.Put("/stores/:storeID/subscription", handler.AdminChangePlan)
	admin.Patch("/stores/:storeID/subscription/status", handler.AdminUpdateStatus)
}
