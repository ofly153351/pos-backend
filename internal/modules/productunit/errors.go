package productunit

import "errors"

var (
	ErrInvalidName          = errors.New("unit name is required")
	ErrProductUnitNotFound  = errors.New("product unit not found")
	ErrProductUnitInUse     = errors.New("product unit is in use by products")
	ErrForbiddenStoreAccess = errors.New("user cannot manage this store")
)
