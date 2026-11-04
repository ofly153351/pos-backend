// Package bizdate is the single definition of a "business date" (B02).
//
// A business date is a calendar day in the store's local timezone
// (Asia/Bangkok), not an instant. Document dates are stored in timestamptz
// columns, and two shapes exist in the data:
//
//   - dates submitted as YYYY-MM-DD, stored as UTC midnight (07:00 in Bangkok);
//   - legacy instants such as a sale's sold_at (e.g. 00:30 Bangkok = 17:30 UTC
//     on the previous UTC day).
//
// Reading the stored instant in Asia/Bangkok yields the intended calendar day
// for both shapes, so no historical data migration is needed. Formatting the
// instant in UTC (or the server/browser zone) is what shifted dates backward.
package bizdate

import (
	"fmt"
	"sync"
	"time"
)

// Layout is the wire format for business dates in requests and query strings.
const Layout = "2006-01-02"

// SQLZone is the zone name used in SQL `AT TIME ZONE` expressions.
const SQLZone = "Asia/Bangkok"

var location = sync.OnceValue(func() *time.Location {
	loc, err := time.LoadLocation(SQLZone)
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
})

// Location returns the business timezone (Asia/Bangkok, fixed +07 fallback).
func Location() *time.Location { return location() }

// FromInstant returns the business date of t as UTC midnight, the canonical
// storage shape for new rows.
func FromInstant(t time.Time) time.Time {
	y, m, d := t.In(Location()).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Today returns today's business date (as UTC midnight).
func Today() time.Time { return FromInstant(time.Now()) }

// Parse parses a YYYY-MM-DD business date into UTC midnight.
func Parse(s string) (time.Time, error) {
	t, err := time.Parse(Layout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid business date %q: %w", s, err)
	}
	return t, nil
}

// Format returns the YYYY-MM-DD business date of a stored value.
func Format(t time.Time) string { return t.In(Location()).Format(Layout) }

// Local returns t in the business timezone, for day/month/year rendering.
func Local(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(Location())
}

// SQLDate returns a SQL expression for the business date of a timestamptz
// column. column must be a trusted identifier, never user input.
func SQLDate(column string) string {
	return "(" + column + " AT TIME ZONE '" + SQLZone + "')::date"
}
