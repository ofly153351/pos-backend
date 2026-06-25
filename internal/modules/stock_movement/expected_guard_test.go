package stock_movement

import (
	"errors"
	"testing"
)

func mkExpected(i int) *int { return &i }

// TestCheckExpectedQuantity locks in the SET_ACTUAL optimistic-lock guard that prevents
// a store-wide aggregate from being written into a single location, and that detects a
// stale count. The guard is the core of the location-aware stock-adjustment remediation.
func TestCheckExpectedQuantity(t *testing.T) {
	cases := []struct {
		name     string
		expected *int
		current  int
		want     error
	}{
		// A SET with no declared expected cannot prove it knows the location's on-hand.
		{"nil expected is rejected", nil, 20, ErrStockExpectedRequired},
		// Caller's expected matches the live location quantity → safe to apply.
		{"matching expected is allowed", mkExpected(20), 20, nil},
		{"zero matches zero", mkExpected(0), 0, nil},
		// The exact reported defect: location holds 20, but a store-wide total of 100 is
		// sent as the absolute. 100 != 20 → blocked (no inflation to 100 in one location).
		{"store-wide total into one location is blocked", mkExpected(100), 20, ErrStockStaleCount},
		// Counter recorded 20 at count time, a sale dropped it to 18 before apply → stale.
		{"count made stale by a sale is blocked", mkExpected(20), 18, ErrStockStaleCount},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkExpectedQuantity(tc.expected, tc.current)
			if !errors.Is(got, tc.want) {
				t.Fatalf("checkExpectedQuantity(%v, %d) = %v, want %v", tc.expected, tc.current, got, tc.want)
			}
		})
	}
}
