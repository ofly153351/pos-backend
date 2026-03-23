package invoice

import "errors"

var (
	ErrInvalidInvoiceItems     = errors.New("invoice items are required")
	ErrInvalidInvoiceItem      = errors.New("each invoice item must include product_id and quantity greater than zero")
	ErrInvalidPaidAmount       = errors.New("paid amount must be greater than zero")
	ErrInvalidPaymentMethod    = errors.New("payment method is required")
	ErrInvalidDiscountType     = errors.New("discount_type must be amount or percent")
	ErrDiscountValueRequired   = errors.New("discount_value is required when discount_type is provided")
	ErrInvalidDiscountValue    = errors.New("discount_value must be greater than or equal to zero")
	ErrInvalidPercentDiscount  = errors.New("percent discount must be between 0 and 100")
	ErrAmountDiscountExceeds   = errors.New("amount discount cannot exceed unit price")
	ErrForbiddenStoreAccess    = errors.New("user cannot operate pos for this store")
	ErrProductNotFound         = errors.New("product not found")
	ErrProductInactive         = errors.New("product is inactive")
	ErrInsufficientStock       = errors.New("insufficient product quantity")
	ErrInvoiceNotFound         = errors.New("invoice not found")
	ErrInvoiceAlreadyPaid      = errors.New("invoice already paid")
	ErrPaymentExceedsRemaining = errors.New("payment exceeds remaining amount")
	ErrCustomerNotFound        = errors.New("customer not found")
)
