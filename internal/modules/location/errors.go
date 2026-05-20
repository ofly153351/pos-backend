package location

import "errors"

var (
	ErrLocationForbidden      = errors.New("user cannot manage this store")
	ErrLocationNotFound       = errors.New("location not found")
	ErrLocationNameRequired   = errors.New("location name is required")
	ErrLocationExists         = errors.New("location with this code already exists in this warehouse")
	ErrInvalidWarehouse       = errors.New("warehouse does not belong to this store")
	ErrLocationInUse          = errors.New("cannot delete location: it has stock records")
)
