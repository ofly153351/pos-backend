package bizdate

import (
	"testing"
	"time"
)

func TestFromInstantUsesBangkokCalendarDay(t *testing.T) {
	// 00:30 Bangkok on 4 Nov is 17:30 UTC on 3 Nov: the business date is 4 Nov.
	soldAt := time.Date(2026, 11, 3, 17, 30, 0, 0, time.UTC)
	got := FromInstant(soldAt)
	want := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("FromInstant(%s) = %s, want %s", soldAt, got, want)
	}
	if Format(soldAt) != "2026-11-04" {
		t.Fatalf("Format(%s) = %s, want 2026-11-04", soldAt, Format(soldAt))
	}
}

func TestUTCMidnightKeepsItsDay(t *testing.T) {
	// Dates submitted as YYYY-MM-DD are stored as UTC midnight; reading them in
	// Bangkok (07:00) must not move the day in either direction.
	d, err := Parse("2026-11-04")
	if err != nil {
		t.Fatal(err)
	}
	if Format(d) != "2026-11-04" || Local(d).Day() != 4 {
		t.Fatalf("UTC midnight rendered as %s / day %d", Format(d), Local(d).Day())
	}
	if !FromInstant(d).Equal(d) {
		t.Fatalf("FromInstant must be idempotent for stored business dates")
	}
}

func TestEndOfBangkokDay(t *testing.T) {
	// 23:59 Bangkok on 4 Nov is 16:59 UTC on 4 Nov.
	late := time.Date(2026, 11, 4, 16, 59, 0, 0, time.UTC)
	if Format(late) != "2026-11-04" {
		t.Fatalf("Format(%s) = %s", late, Format(late))
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, s := range []string{"", "2026-13-01", "04/11/2026", "2026-11-04T00:00:00Z"} {
		if _, err := Parse(s); err == nil {
			t.Fatalf("Parse(%q) accepted", s)
		}
	}
}

func TestLocalZero(t *testing.T) {
	if !Local(time.Time{}).IsZero() {
		t.Fatal("zero time must stay zero")
	}
}

func TestSQLDate(t *testing.T) {
	if got := SQLDate("document_date"); got != "(document_date AT TIME ZONE 'Asia/Bangkok')::date" {
		t.Fatalf("SQLDate = %s", got)
	}
}
