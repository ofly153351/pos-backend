package doccopy

import "testing"

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
