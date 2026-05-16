package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/warehouse"
)

func newWarehouseHandler(db *gorm.DB) warehouse.Handler {
	repo := warehouse.NewPostgresRepository(db)
	service := warehouse.NewService(repo)
	return warehouse.NewHandler(service)
}

func registerWarehouseRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.warehouseHandler.(warehouse.Handler)

	// Warehouse CRUD
	protected.Get("/stores/:storeID/warehouses", handler.ListByStore)
	protected.Get("/stores/:storeID/warehouses/:warehouseID", handler.GetByID)
	protected.Post("/stores/:storeID/warehouses", handler.Create)
	protected.Put("/stores/:storeID/warehouses/:warehouseID", handler.Update)
	protected.Delete("/stores/:storeID/warehouses/:warehouseID", handler.Delete)

	// Warehouse-Product association
	protected.Get("/stores/:storeID/warehouses/:warehouseID/products", handler.ListProducts)
	protected.Post("/stores/:storeID/warehouses/:warehouseID/products", handler.AddProduct)
	protected.Delete("/stores/:storeID/warehouses/:warehouseID/products/:productID", handler.RemoveProduct)
}
