package productunit

import "errors"

var (
	ErrInvalidName          = errors.New("unit name is required")
	ErrProductUnitNotFound  = errors.New("product unit not found")
	ErrForbiddenStoreAccess = errors.New("user cannot manage this store")
)
