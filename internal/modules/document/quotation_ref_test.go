package document

import (
	"errors"
	"testing"
)

// quotationRefNoOf decides what a source document contributes to the invoice's
// "อ้างอิงใบเสนอราคา" row: only a QUOTATION contributes its number.
func TestQuotationRefNoOf(t *testing.T) {
	full := "QUO256909-0009"
	srcID := "doc-src-1"

	invoice := &Document{ID: "inv-1", Type: TypeInvoice, DocumentNoFull: "INV256909-0006", SourceDocumentID: &srcID}
	if got := quotationRefNoOf(invoice); got != "" {
		t.Errorf("invoice source must not produce a quotation ref, got %q", got)
	}
	if got := quotationRefNoOf(&Document{Type: TypeDeliveryOrder, DocumentNoFull: "DO256909-0001"}); got != "" {
		t.Errorf("delivery order source must not produce a quotation ref, got %q", got)
	}
	if got := quotationRefNoOf(nil); got != "" {
		t.Errorf("nil source must produce no ref, got %q", got)
	}
	if got := quotationRefNoOf(&Document{Type: TypeQuotation, DocumentNoFull: full}); got != full {
		t.Errorf("expected %q, got %q", full, got)
	}
	// Legacy rows created before document_no_full existed fall back to document_no.
	if got := quotationRefNoOf(&Document{Type: TypeQuotation, DocumentNo: "QUO256909-0010"}); got != "QUO256909-0010" {
		t.Errorf("expected fallback to document_no, got %q", got)
	}
	// Blank numbers render nothing rather than an empty row.
	if got := quotationRefNoOf(&Document{Type: TypeQuotation, DocumentNoFull: "  "}); got != "" {
		t.Errorf("blank quotation number must produce no ref, got %q", got)
	}
}

// stubRepo implements Repository with only FindByID meaningful — enough to lock the
// render-time wiring (SourceDocumentID → quotation row) without a database.
type stubRepo struct {
	byID map[string]*Document
	err  error
}

func (r stubRepo) FindByID(id string) (*Document, error) {
	if r.err != nil {
		return nil, r.err
	}
	if d, ok := r.byID[id]; ok {
		return d, nil
	}
	return nil, errors.New("not found")
}

func (stubRepo) List(ListQuery) ([]DocumentListItem, int64, DocumentStats, error) {
	return nil, 0, DocumentStats{}, nil
}
func (stubRepo) Create(*Document) error                               { return nil }
func (stubRepo) UpdateStatus(string, DocumentStatus) error            { return nil }
func (stubRepo) MarkPaid(string) error                                { return nil }
func (stubRepo) SetPaymentStatus(string, PaymentStatus) error         { return nil }
func (stubRepo) Delete(string) error                                  { return nil }
func (stubRepo) BulkDelete(string, []string) error                    { return nil }
func (stubRepo) BulkSetStatus(string, []string, DocumentStatus) error { return nil }
func (stubRepo) NextSeq(string, DocumentType) (int64, error)          { return 1, nil }

func TestResolveQuotationRef(t *testing.T) {
	quoID, otherID, missingID := "doc-quo", "doc-sale", "doc-missing"
	svc := Service{repo: stubRepo{byID: map[string]*Document{
		quoID:   {ID: quoID, Type: TypeQuotation, DocumentNoFull: "QUO256909-0009"},
		otherID: {ID: otherID, Type: TypeReceipt, DocumentNoFull: "RCT256909-0001"},
	}}}

	fromQuotation := &Document{ID: "inv-1", Type: TypeInvoice, SourceDocumentID: &quoID}
	if got := svc.resolveQuotationRef(fromQuotation); got != "QUO256909-0009" {
		t.Errorf("expected quotation number, got %q", got)
	}

	fromReceipt := &Document{ID: "inv-2", Type: TypeInvoice, SourceDocumentID: &otherID}
	if got := svc.resolveQuotationRef(fromReceipt); got != "" {
		t.Errorf("non-quotation source must produce no ref, got %q", got)
	}

	noSource := &Document{ID: "inv-3", Type: TypeInvoice}
	if got := svc.resolveQuotationRef(noSource); got != "" {
		t.Errorf("document without a source must produce no ref, got %q", got)
	}

	dangling := &Document{ID: "inv-4", Type: TypeInvoice, SourceDocumentID: &missingID}
	if got := svc.resolveQuotationRef(dangling); got != "" {
		t.Errorf("dangling source must degrade to no ref (logged), got %q", got)
	}
}
