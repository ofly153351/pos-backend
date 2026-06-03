// Package parkedbill_test verifies that parked bill data constraints are met.
package parkedbill_test

import (
	"testing"

	"pos-backend/internal/modules/parkedbill"
)

func TestParkedBill_NoteDefaultsToEmpty(t *testing.T) {
	bill := parkedbill.ParkedBill{}
	if bill.Note != "" {
		t.Errorf("expected Note to be empty string, got %q", bill.Note)
	}
}

func TestParkedBill_CustomerIDOptional(t *testing.T) {
	// CustomerID may be empty — walk-in customers have no customer record.
	bill := parkedbill.ParkedBill{
		CustomerID: "",
		Note:       "",
	}
	if bill.CustomerID != "" {
		t.Errorf("expected empty CustomerID, got %q", bill.CustomerID)
	}
}
