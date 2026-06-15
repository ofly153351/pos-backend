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
	ErrAlreadyCancelled   = errors.New("รายการเครดิตนี้ถูกยกเลิกแล้ว")
	ErrInvalidAmount      = errors.New("payment amount must be greater than zero")
	ErrOverpayment        = errors.New("payment exceeds the outstanding balance")
	// Phase W5 — AddPayment and Cancel now serialize on the credit_sales header row
	// (SELECT ... FOR UPDATE). A payment attempted against an already-cancelled (or
	// concurrently-cancelled) receivable is rejected with this dedicated message.
	ErrPaymentAfterCancel = errors.New("ไม่สามารถรับชำระได้ เนื่องจากรายการเครดิตถูกยกเลิกแล้ว")
	// Phase W5 — cannot cancel a receivable that has collected payment (no reversal model).
	ErrCannotCancelPaid = errors.New("ไม่สามารถยกเลิกรายการเครดิตที่มีการรับชำระแล้วได้")
)
