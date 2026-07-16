package producttype

import "errors"

var (
	ErrInvalidName          = errors.New("product type name is required")
	ErrProductTypeNotFound  = errors.New("product type not found")
	ErrForbiddenStoreAccess = errors.New("user cannot manage this store")
	ErrDuplicateName        = errors.New("name already exists for this store")
)
