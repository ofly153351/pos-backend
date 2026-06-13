package promotion

import "errors"

var (
	ErrStoreIDRequired = errors.New("storeID is required")
	ErrForbidden       = errors.New("user cannot operate this store")
	ErrInvalidBody     = errors.New("invalid promotion payload")
	ErrIDRequired      = errors.New("promotion id is required")
	ErrNotFound        = errors.New("promotion not found")
)
