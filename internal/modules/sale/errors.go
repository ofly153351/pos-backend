package sale

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidSaleItems           = errors.New("sale items are required")
	ErrInvalidSaleItem            = errors.New("each sale item must include product_id and quantity greater than zero")
	ErrInvalidPaymentMethod       = errors.New("payment method is required")
	ErrInvalidPaidAmount          = errors.New("paid amount is less than total amount")
	ErrInvalidBillDiscount        = errors.New("bill discount must be greater than or equal to zero")
	ErrBillDiscountExceedsAmount  = errors.New("bill discount cannot exceed payable amount")
	ErrManualDiscountExceedsCap   = errors.New("manual discount exceeds the allowed cap for your role (cashiers: max 20% of subtotal; ask an owner/manager for a larger discount)")
	ErrInvalidDiscountType        = errors.New("discount_type must be amount or percent")
	ErrDiscountValueRequired      = errors.New("discount_value is required when discount_type is provided")
	ErrInvalidDiscountValue       = errors.New("discount_value must be greater than or equal to zero")
	ErrInvalidPercentDiscount     = errors.New("percent discount must be between 0 and 100")
	ErrInvalidVATPercent          = errors.New("vat_percent must be between 0 and 100")
	ErrAmountDiscountExceedsPrice = errors.New("amount discount cannot exceed unit price")
	ErrForbiddenStoreAccess       = errors.New("user cannot operate pos for this store")
	ErrProductNotFound            = errors.New("product not found")
	ErrProductInactive            = errors.New("product is inactive")
	ErrInsufficientStock          = errors.New("insufficient product quantity")
	ErrSaleNotFound               = errors.New("sale not found")
	ErrCustomerNotFound           = errors.New("customer not found")
	// Phase W4B — location-aware sale deduction (Thai user-facing copy).
	ErrNoSaleLocation          = errors.New("ไม่พบตำแหน่งขายที่พร้อมใช้งาน กรุณากำหนดจุดขายก่อนทำรายการ")
	ErrSaleLocationInvalid     = errors.New("ตำแหน่งที่เลือกไม่ใช่จุดขายที่ใช้งานได้")
	ErrSaleLocationCrossStore  = errors.New("ตำแหน่งที่เลือกไม่อยู่ในร้านเดียวกัน")
	ErrSaleIdempotencyConflict = errors.New("รหัสคำขอนี้ถูกใช้ไปแล้วกับรายการขายที่ไม่ตรงกัน")
)

// InsufficientSaleStockError reports that the resolved sale-point location does not hold
// enough of a product to fulfil the sale. It carries the live remaining quantity so the
// API layer can render the Thai shortfall message with the on-hand count, and unwraps to
// ErrInsufficientStock so existing errors.Is checks keep working. (Phase W4B)
type InsufficientSaleStockError struct {
	ProductID string
	Available int
}

func (e InsufficientSaleStockError) Error() string {
	return fmt.Sprintf("สินค้าในจุดขายมีไม่เพียงพอ คงเหลือ %d รายการ กรุณาโอนสินค้าเข้าจุดขายก่อนขาย", e.Available)
}

func (e InsufficientSaleStockError) Unwrap() error { return ErrInsufficientStock }
