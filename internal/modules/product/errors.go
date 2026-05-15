package product

import "errors"

var (
	ErrInvalidProductName      = errors.New("product name is required")
	ErrInvalidQuantity         = errors.New("quantity must be greater than or equal to zero")
	ErrInvalidBasePrice        = errors.New("base price must be greater than or equal to zero")
	ErrInvalidSpecialPrice     = errors.New("special price must be less than or equal to base price")
	ErrInvalidSpecialPriceDate = errors.New("special price end must be after start")
	ErrInvalidProductTypeID    = errors.New("product type does not belong to this store")
	ErrInvalidProductUnitID    = errors.New("product unit does not belong to this store")
	ErrInvalidPagination       = errors.New("invalid pagination query")
	ErrForbiddenStoreAccess    = errors.New("user cannot manage this store")
	ErrProductNotFound         = errors.New("product not found")
	ErrGenerateSKUFailed       = errors.New("unable to generate unique barcode")
)
