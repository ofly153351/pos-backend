package vat

import "errors"

var (
	ErrNoItemsProvided      = errors.New("items array is required")
	ErrForbiddenStoreAccess = errors.New("user cannot operate pos for this store")
)
