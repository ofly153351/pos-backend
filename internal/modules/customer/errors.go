package customer

import "errors"

var (
	ErrInvalidCustomerName     = errors.New("customer full_name is required")
	ErrInvalidCustomerEmail    = errors.New("valid customer email is required")
	ErrInvalidLevel            = errors.New("level must be greater than zero")
	ErrInvalidDiscountPercent  = errors.New("discount_percent must be between 0 and 100")
	ErrCustomerNotFound        = errors.New("customer not found")
	ErrCustomerStoreIDRequired = errors.New("storeID is required")
	ErrCustomerForbidden       = errors.New("user cannot operate this store")
)
