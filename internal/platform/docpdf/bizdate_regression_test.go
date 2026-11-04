package docpdf

import (
	"testing"
	"time"
)

// B02: see dochtml.TestThaiDateUsesBangkokBusinessDay.
func TestThaiDateUsesBangkokBusinessDay(t *testing.T) {
	soldAt := time.Date(2026, 11, 3, 17, 30, 0, 0, time.UTC)
	if got := thaiDate(soldAt); got != "04/11/2569" {
		t.Fatalf("thaiDate(%s) = %q, want 04/11/2569", soldAt, got)
	}
	if got := thaiDate(time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)); got != "04/11/2569" {
		t.Fatalf("UTC midnight = %q, want 04/11/2569", got)
	}
}
