package dochtml

import (
	"testing"
	"time"
)

// B02: a document created from a sale at 00:30 Bangkok stores 17:30 UTC on the
// previous UTC day. Printed documents must show the Bangkok calendar day.
func TestThaiDateUsesBangkokBusinessDay(t *testing.T) {
	soldAt := time.Date(2026, 11, 3, 17, 30, 0, 0, time.UTC)
	if got := thaiDate(soldAt); got != "4 พฤศจิกายน 2569" {
		t.Fatalf("thaiDate(%s) = %q, want 4 พฤศจิกายน 2569", soldAt, got)
	}
	utcMidnight := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	if got := thaiDate(utcMidnight); got != "4 พฤศจิกายน 2569" {
		t.Fatalf("thaiDate(%s) = %q, want 4 พฤศจิกายน 2569", utcMidnight, got)
	}
}
