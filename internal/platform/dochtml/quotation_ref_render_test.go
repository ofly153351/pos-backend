package dochtml

import (
	"strings"
	"testing"
)

// An invoice created from a quotation must show the source quotation number in the
// header meta table as "อ้างอิงใบเสนอราคา (Ref. Quotation)"; every other document
// (and an invoice without a quotation source) must not grow an empty row.
func TestInvoiceRender_QuotationRefRow(t *testing.T) {
	store := StoreInfo{Name: "ร้านทดสอบ"}

	withRef := makeDoc("INVOICE", 2)
	withRef.QuotationRefNo = "QUO256909-0009"
	html, err := RenderUnifiedDocumentHTML(withRef, store)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	want := `<div class="doc-meta-line"><span class="doc-meta-label">อ้างอิงใบเสนอราคา</span><span class="doc-meta-value">QUO256909-0009</span></div>`
	if !strings.Contains(html, want) {
		t.Errorf("expected quotation ref row %q in invoice HTML", want)
	}

	withoutRef := makeDoc("INVOICE", 2)
	html, err = RenderUnifiedDocumentHTML(withoutRef, store)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if strings.Contains(html, "อ้างอิงใบเสนอราคา") {
		t.Error("quotation ref row must not render when the field is empty")
	}

	// A document that was not created from a quotation (service leaves the field empty)
	// never shows the row — e.g. the quotation itself or a receipt.
	qt := makeDoc("QUOTATION", 2)
	html, err = RenderUnifiedDocumentHTML(qt, store)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if strings.Contains(html, "อ้างอิงใบเสนอราคา") {
		t.Error("quotation ref row must not render on a quotation header")
	}
}

// The print copy set renders the same header per copy — the reference row must survive
// the per-copy render path, not only the single-copy renderer.
func TestInvoiceCopies_QuotationRefRowOnEveryCopy(t *testing.T) {
	doc := makeDoc("INVOICE", 2)
	doc.QuotationRefNo = "QUO256909-0009"

	html, err := RenderUnifiedDocumentCopies(doc, StoreInfo{Name: "ร้านทดสอบ"}, -1)
	if err != nil {
		t.Fatalf("render copies failed: %v", err)
	}
	if got := strings.Count(html, "อ้างอิงใบเสนอราคา"); got == 0 {
		t.Fatal("expected the quotation ref row in the copy set")
	}
	if !strings.Contains(html, `<span class="doc-meta-value">QUO256909-0009</span>`) {
		t.Error("expected the quotation number in the copy set")
	}
}
