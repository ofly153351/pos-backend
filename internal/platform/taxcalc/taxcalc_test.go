package taxcalc

import "testing"

func TestComputeVAT(t *testing.T) {
	cases := []struct {
		name      string
		base      float64
		percent   float64
		included  bool
		wantVAT   float64
		wantTotal float64
	}{
		// T1 — no VAT
		{"zero percent", 100, 0, false, 0, 100},
		{"zero base", 0, 7, false, 0, 0},
		// T6 — exclusive: VAT added on top
		{"exclusive 7%", 100, 7, false, 7, 107},
		{"exclusive 10%", 100, 10, false, 10, 110},
		// T5 — inclusive: VAT carved out, total unchanged
		{"inclusive 7% of 107", 107, 7, true, 7, 107},
		{"inclusive 7% of 100", 100, 7, true, 6.54, 100},
		// T8 — rounding to satang
		{"exclusive rounding", 99.99, 7, false, 7, 106.99},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vat, total := ComputeVAT(c.base, c.percent, c.included)
			if vat != c.wantVAT {
				t.Errorf("vat = %.4f, want %.4f", vat, c.wantVAT)
			}
			if total != c.wantTotal {
				t.Errorf("total = %.4f, want %.4f", total, c.wantTotal)
			}
		})
	}
}

// Guards the EPIC invariant: VAT is always taxed on the post-discount base, never on
// the pre-discount subtotal (the reported Invoice-PDF bug).
func TestComputeVAT_TaxesPostDiscountBase(t *testing.T) {
	// subtotal 100, discount 20 → taxable base 80, exclusive 7%
	vat, total := ComputeVAT(80, 7, false)
	if vat != 5.6 { // 80 * 0.07
		t.Fatalf("vat on discounted base = %.4f, want 5.60 (NOT 7.00 from pre-discount 100)", vat)
	}
	if total != 85.6 {
		t.Fatalf("total = %.4f, want 85.60", total)
	}
}
