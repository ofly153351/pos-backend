package document

import "testing"

func TestBuildReceiptPaymentDescriptionWithBillAndDeliveryRefs(t *testing.T) {
	got := buildReceiptPaymentDescription(
		"BN/2569/09/0001",
		"DO/2569/09/0005",
		"RCT/2569/09/0001",
	)
	want := "ชำระค่าสินค้าตามใบวางบิล เลขที่ BN/2569/09/0001 (ใบส่งสินค้า DO/2569/09/0005)"
	if got != want {
		t.Fatalf("unexpected receipt payment description: got %q, want %q", got, want)
	}
}

func TestBuildReceiptPaymentDescriptionFallsBackWithoutReferences(t *testing.T) {
	got := buildReceiptPaymentDescription("", "", "RCT/2569/09/0001")
	want := "ชำระเงินตามเอกสาร RCT/2569/09/0001"
	if got != want {
		t.Fatalf("unexpected receipt payment fallback: got %q, want %q", got, want)
	}
}
