package sale

import (
	"math"
	"testing"
	"time"
)

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// resolveEffectivePrice is the single source of truth for the per-unit price a sale
// (and therefore any promotion) is computed from: the special price while its window
// is live, else the base price.
func TestResolveEffectivePrice(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	special := 80.0
	before := now.Add(-48 * time.Hour)
	after := now.Add(48 * time.Hour)

	cases := []struct {
		name string
		snap productSnapshot
		want float64
	}{
		{"no special -> base", productSnapshot{BasePrice: 100}, 100},
		{"special active, no bounds -> special", productSnapshot{BasePrice: 100, SpecialPrice: &special}, 80},
		{"special active within window -> special", productSnapshot{BasePrice: 100, SpecialPrice: &special, SpecialPriceStartAt: &before, SpecialPriceEndAt: &after}, 80},
		{"special before start -> base", productSnapshot{BasePrice: 100, SpecialPrice: &special, SpecialPriceStartAt: &after}, 100},
		{"special after end -> base", productSnapshot{BasePrice: 100, SpecialPrice: &special, SpecialPriceEndAt: &before}, 100},
	}
	for _, c := range cases {
		if got := resolveEffectivePrice(c.snap, now); !approxEqual(got, c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

// Regression for the promotion/special-price consistency fix: a percentage promo's
// server-verified ceiling must scale off the EFFECTIVE-price subtotal. Because the
// subtotal handed to promoCeiling is built from resolveEffectivePrice, an item on
// special discounts from the special price — never the full base price. This is the
// backend guard that makes the "no double discount" rule hold even if a client sends
// an inflated promo_discount.
func TestPromoCeilingUsesEffectiveSubtotal(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	special := 80.0
	qty := 1
	specialSnap := productSnapshot{BasePrice: 100, SpecialPrice: &special}
	baseSnap := productSnapshot{BasePrice: 100}

	pct := `{"percentOff":10}`
	specialSubtotal := resolveEffectivePrice(specialSnap, now) * float64(qty) // 80 (special active)
	baseSubtotal := resolveEffectivePrice(baseSnap, now) * float64(qty)       // 100 (no special)

	gotSpecial := promoCeiling("percentage", pct, specialSubtotal, qty)
	gotBase := promoCeiling("percentage", pct, baseSubtotal, qty)

	if !approxEqual(gotSpecial, 8) {
		t.Errorf("10%% promo on special subtotal 80: got %v want 8", gotSpecial)
	}
	if !approxEqual(gotBase, 10) {
		t.Errorf("10%% promo on base subtotal 100: got %v want 10", gotBase)
	}
	if gotSpecial >= gotBase {
		t.Errorf("special-price promo ceiling (%v) must be lower than base-price ceiling (%v)", gotSpecial, gotBase)
	}
}

// The ceiling never exceeds the (effective) subtotal and quantity-dependent types
// derive a sound upper bound — a client cannot inflate a promo to more than the
// effective line value.
func TestPromoCeilingClampAndTypes(t *testing.T) {
	if got := promoCeiling("fixed_amount", `{"amountOff":500}`, 80, 1); !approxEqual(got, 80) {
		t.Errorf("fixed_amount clamps to subtotal: got %v want 80", got)
	}
	if got := promoCeiling("bundle", `{"bundlePrice":60}`, 80, 2); !approxEqual(got, 20) {
		t.Errorf("bundle subtotal-bundlePrice: got %v want 20", got)
	}
	if got := promoCeiling("fixed_price", `{"fixedPrice":30}`, 80, 2); !approxEqual(got, 20) {
		t.Errorf("fixed_price upper bound: got %v want 20", got)
	}
	if got := promoCeiling("mystery_type", `{}`, 80, 1); got != 0 {
		t.Errorf("unknown promo type fails closed: got %v want 0", got)
	}
}
