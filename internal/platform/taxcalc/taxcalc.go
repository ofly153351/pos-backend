// Package taxcalc holds the single, canonical VAT formula shared by every module
// that computes tax (sale, invoice, the standalone /vat/calculate preview). It exists
// so the inclusive/exclusive split lives in exactly one place — callers keep their own
// rounding/default policy around it, but the core arithmetic can never drift between
// what a sale charges, what an invoice records and what a preview shows.
package taxcalc

import "math"

// Round2 rounds a money value to 2 decimal places (satang).
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ComputeVAT derives the VAT amount and the grand total for a taxable base that has
// already had all discounts subtracted.
//
//   - included == true  → the base is VAT-inclusive: VAT is carved OUT of the base and
//     the grand total equals the base (price already contains tax).
//   - included == false → the base is VAT-exclusive: VAT is added ON TOP and the grand
//     total equals base + VAT.
//
// Both returned values are rounded to 2dp. A non-positive percent or base yields zero
// VAT and a grand total equal to the (rounded) base. Callers that collect whole-baht
// cash apply their own final rounding to the returned total.
func ComputeVAT(base, percent float64, included bool) (vat, total float64) {
	if base <= 0 {
		return 0, 0
	}
	if percent <= 0 {
		return 0, Round2(base)
	}
	if included {
		return Round2(base * percent / (100 + percent)), Round2(base)
	}
	vat = Round2(base * percent / 100)
	return vat, Round2(base + vat)
}
