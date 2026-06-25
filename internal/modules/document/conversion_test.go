package document

import "testing"

// The workflow matrix must allow exactly the documented conversions and reject
// everything else (no arbitrary type changes that break accounting flow).
func TestCanConvert_Matrix(t *testing.T) {
	allowed := []struct{ from, to DocumentType }{
		{TypeQuotation, TypeInvoice},
		{TypeInvoice, TypeReceipt},
		{TypeInvoice, TypeTaxInvoice},
		{TypeInvoice, TypeDeliveryOrder},
		{TypeInvoice, TypeCreditNote},
		{TypeReceipt, TypeTaxInvoice},
		{TypeReceipt, TypeCreditNote},
		{TypeDeliveryOrder, TypeInvoice},
		{TypeTaxInvoice, TypeCreditNote},
	}
	for _, c := range allowed {
		if !canConvert(c.from, c.to) {
			t.Errorf("expected %s → %s to be allowed", c.from, c.to)
		}
	}

	denied := []struct{ from, to DocumentType }{
		{TypeQuotation, TypeReceipt},     // quotation can only become an invoice
		{TypeQuotation, TypeCreditNote},  // …
		{TypeReceipt, TypeInvoice},       // receipt cannot revert to invoice
		{TypeTaxInvoice, TypeInvoice},    // tax invoice cannot revert
		{TypeTaxInvoice, TypeReceipt},    // …
		{TypeDeliveryOrder, TypeReceipt}, // DO only becomes an invoice
		{TypeCreditNote, TypeInvoice},    // credit note is terminal
		{TypeInvoice, TypeQuotation},     // cannot go backwards
		{TypeInvoice, TypeInvoice},       // no self-conversion
	}
	for _, c := range denied {
		if canConvert(c.from, c.to) {
			t.Errorf("expected %s → %s to be denied", c.from, c.to)
		}
	}
}
