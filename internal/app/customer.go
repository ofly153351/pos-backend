package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/customer"
)

func newCustomerHandler(db *gorm.DB) customer.Handler {
	repo := customer.NewPostgresRepository(db)
	service := customer.NewService(repo)
	return customer.NewHandler(service)
}

func registerCustomerRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.customerHandler.(customer.Handler)
	protected.Post("/stores/:storeID/customers", handler.Create)
	protected.Get("/stores/:storeID/customers", handler.ListByStore)
	protected.Get("/stores/:storeID/customers/:customerID", handler.GetByID)
	protected.Patch("/stores/:storeID/customers/:customerID", handler.Update)
	protected.Delete("/stores/:storeID/customers/:customerID", handler.Delete)
	protected.Get("/stores/:storeID/customer-level-discounts", handler.ListLevelDiscounts)
	protected.Put("/stores/:storeID/customer-level-discounts/:level", handler.UpsertLevelDiscount)
	protected.Delete("/stores/:storeID/customer-level-discounts/:level", handler.DeleteLevelDiscount)
}
