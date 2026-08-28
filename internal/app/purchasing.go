package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/config"
	"pos-backend/internal/modules/purchasing"
)

func newPurchasingHandler(cfg config.Config, db *gorm.DB) purchasing.Handler {
	repo := purchasing.NewPostgresRepository(db)
	storage := purchasing.NewMinIOSupplierLogoStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucketName,
		cfg.MinIOUseSSL,
		cfg.MinIOPublicURL,
	)
	service := purchasing.NewService(repo, db, storage)
	return purchasing.NewHandler(service)
}

func registerPurchasingRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.purchasingHandler.(purchasing.Handler)
	g := newStoreGuards(deps)

	// Suppliers
	protected.Post("/stores/:storeID/suppliers", g.manage, handler.CreateSupplier)
	protected.Get("/stores/:storeID/suppliers", g.access, handler.ListSuppliers)
	protected.Get("/stores/:storeID/suppliers/:supplierID", g.access, handler.GetSupplier)
	protected.Put("/stores/:storeID/suppliers/:supplierID", g.manage, handler.UpdateSupplier)
	protected.Delete("/stores/:storeID/suppliers/:supplierID", g.manage, handler.DeleteSupplier)

	// Supplier Products
	protected.Get("/stores/:storeID/suppliers/:supplierID/products", g.access, handler.ListSupplierProducts)
	protected.Post("/stores/:storeID/suppliers/:supplierID/products", g.manage, handler.AddSupplierProduct)
	protected.Post("/stores/:storeID/suppliers/:supplierID/products/create", g.manage, handler.CreateSupplierProductWithNewProduct)
	protected.Put("/stores/:storeID/suppliers/:supplierID/products/:productID", g.manage, handler.UpdateSupplierProduct)
	protected.Delete("/stores/:storeID/suppliers/:supplierID/products/:productID", g.manage, handler.RemoveSupplierProduct)

	// Purchase Orders
	protected.Post("/stores/:storeID/purchase-orders", g.manage, handler.CreatePO)
	protected.Get("/stores/:storeID/purchase-orders", g.access, handler.ListPOs)
	protected.Get("/stores/:storeID/purchase-orders/:poID", g.access, handler.GetPO)
	protected.Put("/stores/:storeID/purchase-orders/:poID", g.manage, handler.UpdatePO)
	protected.Post("/stores/:storeID/purchase-orders/:poID/receive", g.access, handler.ReceiveStock)
	protected.Post("/stores/:storeID/purchase-orders/:poID/cancel", g.manage, handler.CancelPO)
}
