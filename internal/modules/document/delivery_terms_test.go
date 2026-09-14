package document

import (
	"testing"
	"time"
)

func TestCalculateTermDates(t *testing.T) {
	docDate := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	poDate := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	validDays, deliveryDays := 30, 15
	validUntil, expected := calculateTermDates(docDate, &validDays, &deliveryDays, &poDate)

	if want := docDate.AddDate(0, 0, 30); !validUntil.Equal(want) {
		t.Fatalf("valid until = %v, want %v", validUntil, want)
	}
	if want := poDate.AddDate(0, 0, 15); !expected.Equal(want) {
		t.Fatalf("expected delivery = %v, want %v", expected, want)
	}
}

func TestCalculateTermDates_BlankPODateLeavesExpectedBlank(t *testing.T) {
	days := 15
	validUntil, expected := calculateTermDates(time.Now(), nil, &days, nil)
	if validUntil != nil || expected != nil {
		t.Fatalf("blank terms should remain blank: valid=%v expected=%v", validUntil, expected)
	}
}
