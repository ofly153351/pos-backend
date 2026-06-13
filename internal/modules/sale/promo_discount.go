package sale

import (
	"encoding/json"
	"math"
	"time"
)

// promoData holds the numeric/schedule fields parsed out of a promotion's `data`
// JSON blob (the full client Campaign object). Only the keys the server needs for
// the discount ceiling + active-window check are declared; the rest are ignored.
type promoData struct {
	PercentOff       float64 `json:"percentOff"`
	AmountOff        float64 `json:"amountOff"`
	DiscountAmount   float64 `json:"discountAmount"`
	ExchangeDiscount float64 `json:"exchangeDiscount"`
	BundlePrice      float64 `json:"bundlePrice"`
	FixedPrice       float64 `json:"fixedPrice"`
	MemberPrice      float64 `json:"memberPrice"`
	BuyQty           float64 `json:"buyQty"`
	GetQty           float64 `json:"getQty"`
	StartDate        string  `json:"startDate"`
	EndDate          string  `json:"endDate"`
}

// promoCeiling returns the theoretical maximum bill discount a single promotion of
// the given type could grant on a cart of `subtotal` containing `totalQty` units.
//
// For the quantity/line-dependent types (fixed_price, member_price, bundle,
// buy_x_get_y) the server cannot re-run the full client engine (no per-line scope),
// so it computes a sound UPPER BOUND from the promotion's own declared parameters
// (assuming, conservatively, that every cart unit is in scope). This is far tighter
// than "the whole subtotal": e.g. a bundle whose bundlePrice >= subtotal yields a 0
// ceiling, so a client cannot inflate promo_discount to 100% off a trivial promo.
// Unknown / non-monetary types fail closed (0).
//
// Field map (keys inside `data`) confirmed against promotion-engine.ts.
func promoCeiling(promoType, data string, subtotal float64, totalQty int) float64 {
	var d promoData
	_ = json.Unmarshal([]byte(data), &d)

	clamp := func(v float64) float64 {
		if v < 0 || math.IsNaN(v) {
			return 0
		}
		if v > subtotal {
			return subtotal
		}
		return v
	}
	qty := float64(totalQty)

	switch promoType {
	case "percentage", "happy_hour":
		p := d.PercentOff
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		return clamp(subtotal * p / 100)
	case "fixed_amount", "coupon":
		return clamp(d.AmountOff)
	case "spend_x_discount":
		return clamp(d.DiscountAmount)
	case "cylinder_exchange":
		return clamp(d.ExchangeDiscount)
	case "bundle":
		// engine: max(0, subtotal - bundlePrice)
		return clamp(subtotal - d.BundlePrice)
	case "fixed_price":
		// engine: max(0, subtotal - fixedPrice*qty); upper bound treats all units as scoped.
		return clamp(subtotal - d.FixedPrice*qty)
	case "member_price":
		// engine: max(0, subtotal - memberPrice*qty); upper bound.
		return clamp(subtotal - d.MemberPrice*qty)
	case "buy_x_get_y":
		// engine: floor(qty/(buy+get)) * get * unitPrice; upper bound with avg unit price.
		group := d.BuyQty + d.GetQty
		if group <= 0 || qty <= 0 {
			return 0
		}
		freeUnits := math.Floor(qty/group) * d.GetQty
		avgUnit := subtotal / qty
		return clamp(freeUnits * avgUnit)
	case "spend_x_gift":
		return 0
	default:
		return 0
	}
}

// promoWithinWindow reports whether `now` is inside the promotion's [startDate,
// endDate] window. Missing bounds are open. A present-but-UNPARSEABLE bound FAILS
// CLOSED (window treated as not satisfied) so an operator cannot neutralise the
// schedule by writing garbage dates. Dates may be bare YYYY-MM-DD (endDate inclusive
// end-of-day) or full RFC3339.
func promoWithinWindow(data string, now time.Time) bool {
	var d promoData
	_ = json.Unmarshal([]byte(data), &d)
	if d.StartDate != "" {
		start, ok := parsePromoDate(d.StartDate, false)
		if !ok || now.Before(start) {
			return false
		}
	}
	if d.EndDate != "" {
		end, ok := parsePromoDate(d.EndDate, true)
		if !ok || now.After(end) {
			return false
		}
	}
	return true
}

// parsePromoDate parses an ISO date/datetime. For a bare date with endOfDay=true the
// returned instant is pushed to the end of that day so the window is inclusive.
func parsePromoDate(value string, endOfDay bool) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		if endOfDay {
			t = t.Add(24*time.Hour - time.Second)
		}
		return t, true
	}
	return time.Time{}, false
}
