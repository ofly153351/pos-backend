package sale

import "errors"

var (
	ErrInvalidSaleItems           = errors.New("sale items are required")
	ErrInvalidSaleItem            = errors.New("each sale item must include product_id and quantity greater than zero")
	ErrInvalidPaymentMethod       = errors.New("payment method is required")
	ErrInvalidPaidAmount          = errors.New("paid amount is less than total amount")
	ErrInvalidDiscountType        = errors.New("discount_type must be amount or percent")
	ErrDiscountValueRequired      = errors.New("discount_value is required when discount_type is provided")
	ErrInvalidDiscountValue       = errors.New("discount_value must be greater than or equal to zero")
	ErrInvalidPercentDiscount     = errors.New("percent discount must be between 0 and 100")
	ErrAmountDiscountExceedsPrice = errors.New("amount discount cannot exceed unit price")
	ErrForbiddenStoreAccess       = errors.New("user cannot operate pos for this store")
	ErrProductNotFound            = errors.New("product not found")
	ErrProductInactive            = errors.New("product is inactive")
	ErrInsufficientStock          = errors.New("insufficient product quantity")
	ErrSaleNotFound               = errors.New("sale not found")
	ErrCustomerNotFound           = errors.New("customer not found")
)
