package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/purchasing"
)

func newPurchasingHandler(db *gorm.DB) purchasing.Handler {
	repo := purchasing.NewPostgresRepository(db)
	service := purchasing.NewService(repo)
	return purchasing.NewHandler(service)
}

func registerPurchasingRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.purchasingHandler.(purchasing.Handler)

	// Suppliers
	protected.Post("/stores/:storeID/suppliers", handler.CreateSupplier)
	protected.Get("/stores/:storeID/suppliers", handler.ListSuppliers)
	protected.Get("/stores/:storeID/suppliers/:supplierID", handler.GetSupplier)
	protected.Put("/stores/:storeID/suppliers/:supplierID", handler.UpdateSupplier)
	protected.Delete("/stores/:storeID/suppliers/:supplierID", handler.DeleteSupplier)

	// Supplier Products
	protected.Get("/stores/:storeID/suppliers/:supplierID/products", handler.ListSupplierProducts)
	protected.Post("/stores/:storeID/suppliers/:supplierID/products", handler.AddSupplierProduct)
	protected.Post("/stores/:storeID/suppliers/:supplierID/products/create", handler.CreateSupplierProductWithNewProduct)
	protected.Put("/stores/:storeID/suppliers/:supplierID/products/:productID", handler.UpdateSupplierProduct)
	protected.Delete("/stores/:storeID/suppliers/:supplierID/products/:productID", handler.RemoveSupplierProduct)

	// Purchase Orders
	protected.Post("/stores/:storeID/purchase-orders", handler.CreatePO)
	protected.Get("/stores/:storeID/purchase-orders", handler.ListPOs)
	protected.Get("/stores/:storeID/purchase-orders/:poID", handler.GetPO)
	protected.Put("/stores/:storeID/purchase-orders/:poID", handler.UpdatePO)
	protected.Post("/stores/:storeID/purchase-orders/:poID/receive", handler.ReceiveStock)
	protected.Post("/stores/:storeID/purchase-orders/:poID/cancel", handler.CancelPO)
}
