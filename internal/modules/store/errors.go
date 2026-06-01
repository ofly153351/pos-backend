package store

import "errors"

var (
	ErrInvalidStoreName         = errors.New("store name is required")
	ErrInvalidCurrencyCode      = errors.New("currency code is required")
	ErrInvalidSubscriptionPlan  = errors.New("subscription plan is required")
	ErrStoreNotFound            = errors.New("store not found")
	ErrSubscriptionPlanNotFound = errors.New("subscription plan not found")
	ErrStoreForbidden           = errors.New("user cannot manage this store")
	ErrUnauthorized             = errors.New("unauthorized")
)
