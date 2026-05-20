package parkedbill

import "errors"

var (
	ErrForbiddenStoreAccess = errors.New("user cannot operate POS for this store")
	ErrParkedBillNotFound   = errors.New("parked bill not found")
	ErrInvalidLabel         = errors.New("label is required")
	ErrInvalidBillDiscount  = errors.New("bill discount must be greater than or equal to zero")
	ErrInvalidVATPercent    = errors.New("vat_percent must be between 0 and 100")
	ErrInvalidItems         = errors.New("at least one item is required")
	ErrInvalidItem          = errors.New("each item must include product_id and quantity greater than zero")
)
