package sale

import "testing"

// Phase W5 regression — the sale idempotency fingerprint must be deterministic for the
// same business intent and must change when any meaningful field changes, so a retried
// Idempotency-Key with a different intent is rejected (409) instead of returning the
// wrong original sale. These are pure-logic tests (no DB, no fixed shared IDs).

func baseReq() CreateSaleRequest {
	return CreateSaleRequest{
		PaymentMethod: "cash",
		LocationID:    "loc-1",
		PaidAmount:    100,
		Note:          "hi",
		CustomerID:    "cust-1",
		Items: []CreateSaleItemRequest{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 1},
		},
	}
}

func TestSaleFingerprint_Deterministic(t *testing.T) {
	a := saleFingerprint("store-1", "user-1", baseReq())
	b := saleFingerprint("store-1", "user-1", baseReq())
	if a != b {
		t.Fatalf("fingerprint not deterministic: %s vs %s", a, b)
	}
}

func TestSaleFingerprint_ItemOrderStable(t *testing.T) {
	r1 := baseReq()
	r2 := baseReq()
	r2.Items = []CreateSaleItemRequest{{ProductID: "p2", Quantity: 1}, {ProductID: "p1", Quantity: 2}}
	if saleFingerprint("s", "u", r1) != saleFingerprint("s", "u", r2) {
		t.Fatal("reordered-but-equivalent cart must produce the same fingerprint")
	}
}

func TestSaleFingerprint_FieldSensitivity(t *testing.T) {
	base := saleFingerprint("store-1", "user-1", baseReq())
	mutations := map[string]func(*CreateSaleRequest, *string, *string){
		"location":       func(r *CreateSaleRequest, _, _ *string) { r.LocationID = "loc-2" },
		"quantity":       func(r *CreateSaleRequest, _, _ *string) { r.Items[0].Quantity = 3 },
		"product":        func(r *CreateSaleRequest, _, _ *string) { r.Items[0].ProductID = "pX" },
		"payment_method": func(r *CreateSaleRequest, _, _ *string) { r.PaymentMethod = "transfer" },
		"paid_amount":    func(r *CreateSaleRequest, _, _ *string) { r.PaidAmount = 200 },
		"note":           func(r *CreateSaleRequest, _, _ *string) { r.Note = "changed" },
		"customer":       func(r *CreateSaleRequest, _, _ *string) { r.CustomerID = "cust-2" },
		"discount_type": func(r *CreateSaleRequest, _, _ *string) {
			dv := 1.0
			r.Items[0].DiscountType = "amount"
			r.Items[0].DiscountValue = &dv
		},
	}
	for name, mut := range mutations {
		r := baseReq()
		mut(&r, nil, nil)
		if got := saleFingerprint("store-1", "user-1", r); got == base {
			t.Errorf("mutation %q did not change the fingerprint (idempotency would wrongly match)", name)
		}
	}
	// The authenticated user is part of the fingerprint: a different user reusing a key conflicts.
	if saleFingerprint("store-1", "user-2", baseReq()) == base {
		t.Error("different authenticated user must change the fingerprint")
	}
	// The store scope is part of the fingerprint.
	if saleFingerprint("store-2", "user-1", baseReq()) == base {
		t.Error("different store must change the fingerprint")
	}
}
