package warehouse

import "errors"

var (
	ErrInvalidName             = errors.New("warehouse name is required")
	ErrWarehouseNotFound       = errors.New("warehouse not found")
	ErrForbiddenStoreAccess    = errors.New("user cannot manage this store")
	ErrProductNotFound         = errors.New("product not found")
	ErrProductNotInWarehouse   = errors.New("product is not assigned to this warehouse")
	ErrTransferInvalidDestination = errors.New("invalid transfer destination type")
	ErrTransferSameWarehouse   = errors.New("source and destination warehouse must be different")
	ErrTransferZeroQty         = errors.New("transfer quantity must be greater than zero")
	ErrInsufficientStock      = errors.New("insufficient stock quantity for transfer")
)
