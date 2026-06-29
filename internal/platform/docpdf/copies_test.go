package docpdf

import (
	"bytes"
	"testing"

	"pos-backend/internal/platform/doccopy"
)

func countPages(pdf []byte) int {
	total := bytes.Count(pdf, []byte("/Type /Page"))
	trees := bytes.Count(pdf, []byte("/Type /Pages"))
	return total - trees
}

func TestRenderInvoicePDFCopies_PageCount(t *testing.T) {
	in := InvoicePDFInput{
		SellerName: "Test Gas", DocTitleTH: "ใบส่งของ / ใบกำกับภาษี", DocTitleEN: "Delivery Note / Tax Invoice",
		InvoiceNo: "DO-1", CustomerName: "ลูกค้า",
		Subtotal: 100, TotalAmount: 107, VATRate: 7, VATAmount: 7,
		Items: []InvoicePDFItem{{Description: "ถังแก๊ส", Quantity: 1, Unit: "ถัง", UnitPrice: 100, LineAmount: 100}},
	}

	single, err := RenderInvoicePDF(in)
	if err != nil {
		t.Fatal(err)
	}
	if p := countPages(single); p != 1 {
		t.Fatalf("single: want 1 page, got %d", p)
	}

	variants := doccopy.SpecFor("DELIVERY_ORDER")
	if len(variants) != 3 {
		t.Fatalf("spec: want 3 variants, got %d", len(variants))
	}
	all, err := RenderInvoicePDFCopies(in, variants)
	if err != nil {
		t.Fatal(err)
	}
	if p := countPages(all); p != 3 {
		t.Fatalf("3-copy: want 3 pages, got %d", p)
	}

	one, err := RenderInvoicePDFCopies(in, variants[0:1])
	if err != nil {
		t.Fatal(err)
	}
	if p := countPages(one); p != 1 {
		t.Fatalf("single-variant: want 1 page, got %d", p)
	}
}
