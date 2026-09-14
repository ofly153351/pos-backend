package document

import (
	"testing"
	"time"
)

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
		{TypeQuotation, TypeReceipt},
		{TypeQuotation, TypeCreditNote},
		{TypeReceipt, TypeInvoice},
		{TypeTaxInvoice, TypeInvoice},
		{TypeTaxInvoice, TypeReceipt},
		{TypeDeliveryOrder, TypeReceipt},
		{TypeCreditNote, TypeInvoice},
		{TypeInvoice, TypeQuotation},
		{TypeInvoice, TypeInvoice},
	}
	for _, c := range denied {
		if canConvert(c.from, c.to) {
			t.Errorf("expected %s → %s to be denied", c.from, c.to)
		}
	}
}

func TestBuildConversionRequest_CopiesQuotationTerms(t *testing.T) {
	validDays, deliveryDays := 30, 15
	poDate := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	validUntil := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	expected := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	src := &Document{
		ID:                   "qt-1",
		Type:                 TypeQuotation,
		CustomerID:           "cus-1",
		PriceValidityDays:    &validDays,
		ValidUntil:           &validUntil,
		DeliveryLeadTimeDays: &deliveryDays,
		POReceivedDate:       &poDate,
		ExpectedDeliveryDate: &expected,
	}
	req := buildConversionRequest(src, TypeInvoice)
	if req.PriceValidityDays != &validDays || req.DeliveryLeadTimeDays != &deliveryDays {
		t.Fatal("conversion did not copy quotation term day pointers")
	}
	if req.POReceivedDate == nil || *req.POReceivedDate != "2026-09-05" {
		t.Fatalf("po received date = %v, want 2026-09-05", req.POReceivedDate)
	}
}
