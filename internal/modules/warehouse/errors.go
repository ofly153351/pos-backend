package warehouse

import "errors"

var (
	ErrInvalidName                  = errors.New("warehouse name is required")
	ErrWarehouseNotFound            = errors.New("warehouse not found")
	ErrForbiddenStoreAccess         = errors.New("user cannot manage this store")
	ErrProductNotFound              = errors.New("product not found")
	ErrProductAlreadyInWarehouse    = errors.New("product already assigned to this warehouse")
	ErrProductNotInWarehouse        = errors.New("product is not assigned to this warehouse")
	ErrStandaloneProductNameRequired = errors.New("product name is required for standalone warehouse products")
)
