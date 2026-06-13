package stockcount

import "errors"

var (
	ErrStoreIDRequired   = errors.New("storeID is required")
	ErrForbidden         = errors.New("user cannot operate this store")
	ErrSessionIDRequired = errors.New("session id is required")
	ErrSessionIDMismatch = errors.New("session id in body does not match the URL")
	ErrSessionNotFound   = errors.New("stock count session not found")

	ErrNoItemsToApply        = errors.New("no items to apply")
	ErrInvalidApplyItem      = errors.New("apply item is invalid")
	ErrSessionAlreadyApplied = errors.New("stock count session already applied")
)
