package document

import "testing"

func TestCanEditDocumentProtectsFinalDocuments(t *testing.T) {
	tests := []struct {
		name string
		doc  Document
		want bool
	}{
		{name: "pending", doc: Document{Status: StatusPending, PaymentStatus: PaymentUnpaid}, want: true},
		{name: "completed", doc: Document{Status: StatusCompleted, PaymentStatus: PaymentUnpaid}, want: false},
		{name: "paid", doc: Document{Status: StatusPending, PaymentStatus: PaymentPaid}, want: false},
		{name: "cancelled", doc: Document{Status: StatusCancelled, PaymentStatus: PaymentUnpaid}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canEditDocument(&tt.doc); got != tt.want {
				t.Fatalf("canEditDocument() = %v, want %v", got, tt.want)
			}
		})
	}
}
