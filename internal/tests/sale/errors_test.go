package sale_test

import (
	"testing"

	"pos-backend/internal/modules/sale"
)

func TestSaleErrors_Distinct(t *testing.T) {
	// Each sentinel error must be a distinct value to prevent mis-routing in writeXxxError.
	errors := []error{
		sale.ErrInvalidSaleItems,
		sale.ErrInvalidSaleItem,
		sale.ErrInvalidPaymentMethod,
		sale.ErrInvalidPaidAmount,
		sale.ErrInvalidBillDiscount,
		sale.ErrBillDiscountExceedsAmount,
		sale.ErrInvalidDiscountType,
		sale.ErrDiscountValueRequired,
		sale.ErrInvalidDiscountValue,
		sale.ErrInvalidPercentDiscount,
		sale.ErrInvalidVATPercent,
		sale.ErrAmountDiscountExceedsPrice,
		sale.ErrForbiddenStoreAccess,
		sale.ErrProductNotFound,
		sale.ErrProductInactive,
		sale.ErrInsufficientStock,
		sale.ErrSaleNotFound,
		sale.ErrCustomerNotFound,
	}

	seen := make(map[string]bool)
	for _, e := range errors {
		msg := e.Error()
		if seen[msg] {
			t.Errorf("duplicate error message: %q", msg)
		}
		seen[msg] = true
	}
}

func TestSaleError_Messages(t *testing.T) {
	cases := []struct {
		err      error
		contains string
	}{
		{sale.ErrInvalidSaleItems, "items"},
		{sale.ErrInvalidPaymentMethod, "payment method"},
		{sale.ErrInvalidPaidAmount, "paid amount"},
		{sale.ErrInsufficientStock, "insufficient"},
	}
	for _, tc := range cases {
		if tc.err.Error() == "" {
			t.Errorf("%v: error message must not be empty", tc.err)
		}
	}
}
