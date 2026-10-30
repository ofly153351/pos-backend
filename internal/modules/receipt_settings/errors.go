package receipt_settings

import "errors"

var (
	ErrForbidden    = errors.New("user cannot manage this store")
	ErrNotFound     = errors.New("receipt settings not found")
	ErrInvalidInput = errors.New("invalid document template settings")
)
