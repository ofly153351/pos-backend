package purchasing_test

import (
	"testing"

	"pos-backend/internal/modules/purchasing"
)

func TestPurchasingErrors_Distinct(t *testing.T) {
	errors := []error{
		purchasing.ErrSupplierNameRequired,
		purchasing.ErrSupplierNotFound,
		purchasing.ErrSupplierForbidden,
		purchasing.ErrPONotFound,
		purchasing.ErrPOForbidden,
		purchasing.ErrPOItemsRequired,
		purchasing.ErrPOInvalidQuantity,
		purchasing.ErrPOInvalidUnitCost,
		purchasing.ErrPOAlreadyCompleted,
		purchasing.ErrPOAlreadyCancelled,
		purchasing.ErrSupplierProductExists,
		purchasing.ErrSupplierProductNotFound,
		purchasing.ErrSupplierProductRequired,
		purchasing.ErrSupplierProductNameReq,
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

func TestPurchasingErrors_NotEmpty(t *testing.T) {
	errors := []error{
		purchasing.ErrSupplierNameRequired,
		purchasing.ErrPOItemsRequired,
		purchasing.ErrPOInvalidQuantity,
	}
	for _, e := range errors {
		if e.Error() == "" {
			t.Errorf("error message must not be empty: %T", e)
		}
	}
}
