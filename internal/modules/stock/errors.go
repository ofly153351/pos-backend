package stock

import "errors"

var (
	ErrStockForbidden         = errors.New("user cannot manage this store")
	ErrStockNotFound          = errors.New("stock not found")
	ErrInsufficientStock      = errors.New("insufficient stock quantity")
	ErrNegativeStock          = errors.New("stock quantity cannot be negative")
	ErrProductNotFound        = errors.New("product not found")
	ErrLocationNotFound       = errors.New("location not found")
)
