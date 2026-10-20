package document

import "testing"

func TestTypePrefix_QuotationUsesQT(t *testing.T) {
	if got := typePrefix(TypeQuotation); got != "QT" {
		t.Fatalf("quotation prefix = %q, want QT", got)
	}
}
