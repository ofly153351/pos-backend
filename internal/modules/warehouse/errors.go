package warehouse

import "errors"

var (
	ErrInvalidName                  = errors.New("warehouse name is required")
	ErrWarehouseNotFound            = errors.New("warehouse not found")
	ErrForbiddenStoreAccess         = errors.New("user cannot manage this store")
	ErrProductNotFound              = errors.New("product not found")
	ErrProductNotInWarehouse        = errors.New("product is not assigned to this warehouse")
	ErrTransferInvalidDestination   = errors.New("invalid transfer destination type")
	ErrTransferSameWarehouse        = errors.New("source and destination warehouse must be different")
	ErrTransferZeroQty              = errors.New("transfer quantity must be greater than zero")
	ErrInsufficientStock            = errors.New("insufficient stock quantity for transfer")
	ErrInventoryNotFound            = errors.New("warehouse inventory record not found")
	ErrInventoryInsufficientQty     = errors.New("insufficient inventory quantity for allocation")
	ErrAllocateZeroQty              = errors.New("allocation quantity must be greater than zero")
	ErrAllocateExceedsQty           = errors.New("allocation quantity exceeds available inventory")
	// Phase W0 safety guards.
	ErrWarehouseInUse               = errors.New("cannot delete warehouse: it still has locations, stock, or related records")
	ErrProductHasStock              = errors.New("cannot remove product from warehouse while stock remains")
	ErrWarehouseDirectStockDisabled = errors.New("direct warehouse stock changes are disabled; use goods receiving or the inventory stock adjustment")
)
