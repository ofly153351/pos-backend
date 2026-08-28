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
	g := newStoreGuards(deps)
	protected.Post("/stores/:storeID/customers", g.operate, handler.Create)
	protected.Get("/stores/:storeID/customers", g.operate, handler.ListByStore)
	protected.Get("/stores/:storeID/customers/:customerID", g.operate, handler.GetByID)
	protected.Patch("/stores/:storeID/customers/:customerID", g.operate, handler.Update)
	protected.Delete("/stores/:storeID/customers/:customerID", g.operate, handler.Delete)
	protected.Get("/stores/:storeID/customers/:customerID/shipping-addresses", g.operate, handler.ListShippingAddresses)
	protected.Post("/stores/:storeID/customers/:customerID/shipping-addresses", g.operate, handler.CreateShippingAddress)
	protected.Put("/stores/:storeID/customers/:customerID/shipping-addresses/:addrID", g.operate, handler.UpdateShippingAddress)
	protected.Delete("/stores/:storeID/customers/:customerID/shipping-addresses/:addrID", g.operate, handler.DeleteShippingAddress)
	protected.Get("/stores/:storeID/customer-level-discounts", g.operate, handler.ListLevelDiscounts)
	protected.Put("/stores/:storeID/customer-level-discounts/:level", g.manage, handler.UpsertLevelDiscount)
	protected.Delete("/stores/:storeID/customer-level-discounts/:level", g.manage, handler.DeleteLevelDiscount)
}
