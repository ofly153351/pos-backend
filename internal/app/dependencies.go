package app

import (
	"gorm.io/gorm"
	"pos-backend/internal/config"
)

func newDependencies(cfg config.Config, db *gorm.DB) appDependencies {
	tokenManager, authHandler := newAuthDependencies(cfg, db)

	return appDependencies{
		tokenManager:        tokenManager,
		authHandler:         authHandler,
		storeHandler:        newStoreHandler(cfg, db),
		productTypeHandler:  newProductTypeHandler(db),
		productUnitHandler:  newProductUnitHandler(db),
		productHandler:      newProductHandler(cfg, db),
		saleHandler:         newSaleHandler(db),
		subscriptionHandler: newSubscriptionHandler(db),
	}
}
