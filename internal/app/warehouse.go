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
	protected.Put("/stores/:storeID/warehouses/:warehouseID/products/:productID", handler.UpdateProduct)
	protected.Delete("/stores/:storeID/warehouses/:warehouseID/products/:productID", handler.RemoveProduct)

	// Warehouse Transfer
	protected.Post("/stores/:storeID/warehouses/:warehouseID/transfer", handler.TransferStock)

	// Warehouse-scoped product inventory (read-only; พร้อมขาย/พื้นที่จัดเก็บ/รวมในคลัง split).
	// Registered before the :productID route below so the static "products" segment is unambiguous.
	protected.Get("/stores/:storeID/warehouses/:warehouseID/inventory/products", handler.ListInventoryProducts)

	// Warehouse Inventory (cross-store transfers awaiting allocation)
	protected.Get("/stores/:storeID/warehouses/:warehouseID/inventory", handler.ListInventory)
	protected.Post("/stores/:storeID/warehouses/:warehouseID/inventory/:productID/allocate", handler.AllocateInventory)
}
