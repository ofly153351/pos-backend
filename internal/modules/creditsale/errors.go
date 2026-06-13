package creditsale

import "errors"

var (
	ErrStoreIDRequired    = errors.New("storeID is required")
	ErrForbidden          = errors.New("user cannot operate this store")
	ErrCustomerRequired   = errors.New("a customer is required for a credit sale")
	ErrNoItems            = errors.New("at least one item is required")
	ErrInvalidItem        = errors.New("each item needs a product_id and a positive quantity")
	ErrInvalidDownPayment = errors.New("down payment must be between 0 and the total")
	ErrInvalidType        = errors.New("type must be 'credit' or 'loan'")
	ErrNotFound           = errors.New("credit sale not found")
	ErrAlreadyCancelled   = errors.New("credit sale is already cancelled")
	ErrInvalidAmount      = errors.New("payment amount must be greater than zero")
	ErrOverpayment        = errors.New("payment exceeds the outstanding balance")
)
