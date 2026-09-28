package doccopy

import "testing"

func TestTaxInvoice_HasCustomerAccountingAndBillingCopies(t *testing.T) {
	variants := SpecFor("TAX_INVOICE")
	if len(variants) != 3 {
		t.Fatalf("expected 3 tax-invoice copies, got %d", len(variants))
	}
	want := []string{
		"(สำหรับลูกค้า)",
		"(สำหรับบริษัท — ฝ่ายบัญชีบริษัท)",
		"(สำหรับติดตามการชำระเงิน — วางบิล)",
	}
	for i, variant := range variants {
		if variant.Purpose != want[i] {
			t.Errorf("copy %d purpose = %q, want %q", i, variant.Purpose, want[i])
		}
		if !variant.ShowSignature {
			t.Errorf("copy %d must show the signature block", i)
		}
	}
}
func TestDeliveryOrder_EveryCopyShowsSignature(t *testing.T) {
	variants := SpecFor("DELIVERY_ORDER")
	if len(variants) != 3 {
		t.Fatalf("expected 3 delivery-order copies, got %d", len(variants))
	}
	for i, variant := range variants {
		if !variant.ShowSignature {
			t.Errorf("copy %d (%s) must show the delivered-by/received-by signature block", i, variant.Purpose)
		}
	}
}
