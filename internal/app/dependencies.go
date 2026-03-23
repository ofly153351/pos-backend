package app

import (
	"database/sql"

	"pos-backend/internal/config"
)

func newDependencies(cfg config.Config, db *sql.DB) appDependencies {
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
