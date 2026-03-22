package subscription

import "errors"

var (
	ErrInvalidPlanCode      = errors.New("plan code is required")
	ErrPlanNotFound         = errors.New("subscription plan not found")
	ErrForbiddenStoreAccess = errors.New("user cannot manage this store")
	ErrSubscriptionNotFound = errors.New("store subscription not found")
	ErrInvalidStatus        = errors.New("subscription status is invalid")
)
