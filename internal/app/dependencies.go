package app

import (
	"gorm.io/gorm"

	"pos-backend/internal/config"
)

func newDependencies(cfg config.Config, db *gorm.DB) appDependencies {
	tokenManager, authRepo, authHandler := newAuthDependencies(cfg, db)

	return appDependencies{
		tokenManager:              tokenManager,
		authUserRepo:              authRepo,
		authHandler:               authHandler,
		storeHandler:              newStoreHandler(cfg, db),
		productTypeHandler:        newProductTypeHandler(db),
		productUnitHandler:        newProductUnitHandler(db),
		productBrandHandler:       newProductBrandHandler(db),
		productHandler:            newProductHandler(cfg, db),
		customerHandler:           newCustomerHandler(db),
		vatHandler:                newVATHandler(db),
		saleHandler:               newSaleHandler(db),
		parkedBillHandler:         newParkedBillHandler(db),
		invoiceHandler:            newInvoiceHandler(cfg, db),
		subscriptionHandler:       newSubscriptionHandler(db),
		dashboardHandler:          newDashboardHandler(db),
		warehouseHandler:          newWarehouseHandler(db),
		purchasingHandler:         newPurchasingHandler(cfg, db),
		stockMovementHandler:      newStockMovementHandler(db),
		stockHandler:              newStockHandler(db),
		locationHandler:           newLocationHandler(db),
		warehouseDashboardHandler: newWarehouseDashboardHandler(db),
		warehouseReceiptHandler:   newWarehouseReceiptHandler(cfg, db),
		receiptSettingsHandler:    newReceiptSettingsHandler(db),
	}
}
