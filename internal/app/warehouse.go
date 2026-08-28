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
	g := newStoreGuards(deps)

	// Warehouse CRUD
	protected.Get("/stores/:storeID/warehouses", g.manage, handler.ListByStore)
	protected.Get("/stores/:storeID/warehouses/:warehouseID", g.manage, handler.GetByID)
	protected.Post("/stores/:storeID/warehouses", g.manage, handler.Create)
	protected.Put("/stores/:storeID/warehouses/:warehouseID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/warehouses/:warehouseID", g.manage, handler.Delete)

	// Safe-delete lifecycle: read-only assessment that drives the adaptive delete/archive
	// modal. The static "deletion-assessment" segment follows the :warehouseID capture, so
	// it does not collide with the GetByID route above.
	protected.Get("/stores/:storeID/warehouses/:warehouseID/deletion-assessment", g.manage, handler.AssessDeletion)

	// Warehouse-Product association
	protected.Get("/stores/:storeID/warehouses/:warehouseID/products", g.manage, handler.ListProducts)
	protected.Post("/stores/:storeID/warehouses/:warehouseID/products", g.manage, handler.AddProduct)
	protected.Put("/stores/:storeID/warehouses/:warehouseID/products/:productID", g.manage, handler.UpdateProduct)
	protected.Delete("/stores/:storeID/warehouses/:warehouseID/products/:productID", g.manage, handler.RemoveProduct)

	// Warehouse Transfer
	protected.Post("/stores/:storeID/warehouses/:warehouseID/transfer", g.manage, handler.TransferStock)

	// Warehouse-scoped product inventory (read-only; พร้อมขาย/พื้นที่จัดเก็บ/รวมในคลัง split).
	// Registered before the :productID route below so the static "products" segment is unambiguous.
	protected.Get("/stores/:storeID/warehouses/:warehouseID/inventory/products", g.access, handler.ListInventoryProducts)

	// Warehouse Inventory (cross-store transfers awaiting allocation)
	protected.Get("/stores/:storeID/warehouses/:warehouseID/inventory", g.manage, handler.ListInventory)
	protected.Post("/stores/:storeID/warehouses/:warehouseID/inventory/:productID/allocate", g.manage, handler.AllocateInventory)
}
