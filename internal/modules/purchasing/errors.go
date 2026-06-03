package purchasing

import "errors"

var (
	ErrSupplierNameRequired    = errors.New("supplier name is required")
	ErrSupplierNotFound        = errors.New("supplier not found")
	ErrSupplierForbidden       = errors.New("user cannot manage suppliers in this store")
	ErrSupplierStoreIDReq      = errors.New("storeID is required for supplier")
	ErrPONotFound              = errors.New("purchase order not found")
	ErrPOForbidden             = errors.New("user cannot manage purchase orders in this store")
	ErrPOStoreIDRequired       = errors.New("storeID is required for purchase order")
	ErrPOInvalidStatus         = errors.New("invalid purchase order status")
	ErrPOAlreadyCompleted      = errors.New("purchase order is already completed")
	ErrPOAlreadyCancelled      = errors.New("purchase order is already cancelled")
	ErrPOItemsRequired         = errors.New("purchase order must have at least one item")
	ErrPOInvalidQuantity       = errors.New("quantity must be greater than zero")
	ErrPOInvalidUnitCost       = errors.New("unit cost must be greater than or equal to zero")
	ErrPOProductNotFound       = errors.New("product not found")
	ErrPOReceiveInvalidQty     = errors.New("received quantity cannot exceed ordered quantity")
	ErrPOOrderNumberGenerate   = errors.New("failed to generate order number")
	ErrSupplierProductExists   = errors.New("product already linked to this supplier")
	ErrSupplierProductNotFound = errors.New("supplier product link not found")
	ErrSupplierProductRequired = errors.New("product_id is required")
	ErrSupplierProductNameReq  = errors.New("product name is required")
)
