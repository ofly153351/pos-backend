package productbrand

import "errors"

var (
	ErrInvalidName          = errors.New("product brand name is required")
	ErrProductBrandNotFound = errors.New("product brand not found")
	ErrForbiddenStoreAccess = errors.New("user cannot manage this store")
	ErrDuplicateName        = errors.New("name already exists for this store")
)
