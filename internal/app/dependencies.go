package app

import (
	"gorm.io/gorm"
	"pos-backend/internal/config"
)

func newDependencies(cfg config.Config, db *gorm.DB) appDependencies {
	tokenManager, authRepo, authHandler := newAuthDependencies(cfg, db)

	return appDependencies{
		tokenManager:        tokenManager,
		authUserRepo:        authRepo,
		authHandler:         authHandler,
		storeHandler:        newStoreHandler(cfg, db),
		productTypeHandler:  newProductTypeHandler(db),
		productUnitHandler:  newProductUnitHandler(db),
		productHandler:      newProductHandler(cfg, db),
		customerHandler:     newCustomerHandler(db),
		vatHandler:          newVATHandler(),
		saleHandler:         newSaleHandler(db),
		invoiceHandler:      newInvoiceHandler(db),
		subscriptionHandler: newSubscriptionHandler(db),
	}
}
