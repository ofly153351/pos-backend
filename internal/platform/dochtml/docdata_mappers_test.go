package dochtml

import (
	"math"
	"testing"
)

// =============================================================================
//  docdata_mappers_test.go — verified passing (go1.22)
//  รัน:  go test ./internal/platform/dochtml/ -run VAT -v
// =============================================================================

// Invariant: Subtotal − Discount + Vat == Total, ทุกค่า ≥ 0, rate=0 → vat=0
func TestVAT_Invariants(t *testing.T) {
	lines := []SaleLineInput{
		{SKU: "A", Quantity: 50, UnitPriceInclVat: 663.4},
		{SKU: "B", Quantity: 5, UnitPriceInclVat: 909.5, LineDiscountInclVat: 100},
		{SKU: "C", Quantity: 30, UnitPriceInclVat: 16.05},
	}
	for _, rate := range []float64{0, 7, 10, 7.5} {
		for _, od := range []float64{0, 100, 4280} {
			_, b := convertInclusiveToExclusive(lines, rate, od)
			base := round2(b.Subtotal - b.Discount)
			if math.Abs((base+b.Vat)-b.Total) > 0.001 {
				t.Fatalf("rate=%g od=%g: base+vat=%.2f != total %.2f", rate, od, base+b.Vat, b.Total)
			}
			if b.Subtotal < 0 || b.Vat < 0 || b.Total < 0 {
				t.Fatalf("rate=%g od=%g: negative %+v", rate, od, b)
			}
			if rate == 0 && b.Vat != 0 {
				t.Fatalf("rate=0 ต้องได้ vat=0 แต่ได้ %.2f", b.Vat)
			}
		}
	}
}

// Known value: 3 บรรทัด รวม VAT บรรทัดละ 107 @ 7% → ex 100 ต่อบรรทัด
func TestVAT_KnownValue(t *testing.T) {
	lines := []SaleLineInput{
		{Quantity: 1, UnitPriceInclVat: 107},
		{Quantity: 1, UnitPriceInclVat: 107},
		{Quantity: 1, UnitPriceInclVat: 107},
	}
	items, b := convertInclusiveToExclusive(lines, 7, 0)
	if b.Subtotal != 300 {
		t.Fatalf("subtotal got %.2f want 300", b.Subtotal)
	}
	if math.Abs(b.Vat-21) > 0.01 {
		t.Fatalf("vat got %.2f want ~21", b.Vat)
	}
	if math.Abs(b.Total-321) > 0.01 {
		t.Fatalf("total got %.2f want ~321", b.Total)
	}
	if math.Abs(items[0].UnitPrice-100) > 0.01 {
		t.Fatalf("exclusive unit price got %.2f want 100", items[0].UnitPrice)
	}
}

// rate=0: exclusive ต้องเท่ากับ inclusive ทุกประการ
func TestVAT_NoVatPassthrough(t *testing.T) {
	lines := []SaleLineInput{{Quantity: 3, UnitPriceInclVat: 250, LineDiscountInclVat: 50}}
	items, b := convertInclusiveToExclusive(lines, 0, 0)
	if items[0].UnitPrice != 250 || items[0].Amount != 700 { // 3*250-50
		t.Fatalf("passthrough ผิด: unit=%.2f amount=%.2f", items[0].UnitPrice, items[0].Amount)
	}
	if b.Subtotal != 700 || b.Total != 700 {
		t.Fatalf("passthrough totals ผิด: %+v", b)
	}
}
