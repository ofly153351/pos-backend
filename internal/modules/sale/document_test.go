package sale

import (
	"strings"
	"testing"
	"time"

	"pos-backend/internal/platform/dochtml"
)

func sampleSaleForDoc() Sale {
	return Sale{
		ID:                 "sale-1",
		SaleNumber:         "SALE-20260622-0001",
		CashierName:        "Owner",
		CustomerName:       "บริษัท ทดสอบ จำกัด",
		CustomerPhone:      "021234567",
		StoreName:          "ร้านทดสอบ",
		StoreAddress:       "123 ถนนทดสอบ",
		StorePhone:         "020000000",
		StoreTaxID:         "0105500000000",
		SubtotalAmount:     1000,
		// DiscountAmount is the COMBINED item + bill discount, exactly as the sale
		// repository stores it (item 50 + bill 50 = 100).
		DiscountAmount:     100,
		BillDiscountAmount: 50, // bill-level (already included in DiscountAmount)
		VATPercent:         7,
		VATAmount:          63,
		TotalAmount:        963,
		PaidAmount:         1000,
		ChangeAmount:       37,
		SoldAt:             time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC),
		Items: []SaleItem{
			{ProductName: "สินค้า A", SKU: "A001", UnitType: "ชิ้น", Quantity: 2, UnitPrice: 300, LineDiscountTotal: 50, LineTotal: 550},
			{ProductName: "สินค้า B", SKU: "B002", UnitType: "ชิ้น", Quantity: 1, UnitPrice: 400, LineDiscountTotal: 0, LineTotal: 400},
		},
	}
}

func TestBuildSaleDocData(t *testing.T) {
	s := sampleSaleForDoc()
	d := buildSaleDocData(s, "TAX_INVOICE", 7, 63)

	if d.Type != "TAX_INVOICE" {
		t.Errorf("Type = %q, want TAX_INVOICE", d.Type)
	}
	if d.DocumentNo != s.SaleNumber || d.DocumentNoFull != s.SaleNumber {
		t.Errorf("DocumentNo mismatch: %q / %q", d.DocumentNo, d.DocumentNoFull)
	}
	if len(d.Items) != 2 {
		t.Fatalf("Items = %d, want 2", len(d.Items))
	}
	if d.Items[0].Description != "สินค้า A" || d.Items[0].SKU != "A001" || d.Items[0].Quantity != 2 {
		t.Errorf("item[0] mapping wrong: %+v", d.Items[0])
	}
	// TotalDiscount is the combined discount (DiscountAmount) — NOT item + bill added
	// again (that double-counted the bill discount vs the receipt).
	if d.TotalDiscount != 100 {
		t.Errorf("TotalDiscount = %v, want 100", d.TotalDiscount)
	}
	// PreVatAmount is net of VAT.
	if d.PreVatAmount != 900 {
		t.Errorf("PreVatAmount = %v, want 900", d.PreVatAmount)
	}
	if d.VatAmount != 63 || d.TotalAmount != 963 || d.Subtotal != 1000 {
		t.Errorf("amounts wrong: pre=%v vat=%v total=%v sub=%v", d.PreVatAmount, d.VatAmount, d.TotalAmount, d.Subtotal)
	}
	if d.StaffName != "Owner" || d.CustomerName != "บริษัท ทดสอบ จำกัด" {
		t.Errorf("party mapping wrong: staff=%q customer=%q", d.StaffName, d.CustomerName)
	}
}

func TestBuildSaleDocDataNoteOptional(t *testing.T) {
	s := sampleSaleForDoc()
	if d := buildSaleDocData(s, "INVOICE", 7, 63); d.Notes != nil {
		t.Errorf("empty note should map to nil, got %v", *d.Notes)
	}
	s.Note = "ด่วน"
	if d := buildSaleDocData(s, "INVOICE", 7, 63); d.Notes == nil || *d.Notes != "ด่วน" {
		t.Errorf("note not mapped")
	}
}

// documentVAT mirrors the receipt: VAT only breaks out for an exclusive-tax store.
func TestDocumentVAT(t *testing.T) {
	s := sampleSaleForDoc() // VATPercent 7, VATAmount 63
	if r, a := documentVAT(s, ReceiptSettingsView{TaxMode: "exclusive"}); r != 7 || a != 63 {
		t.Errorf("exclusive: got rate=%v amount=%v, want 7/63", r, a)
	}
	if r, a := documentVAT(s, ReceiptSettingsView{TaxMode: "inclusive"}); r != 0 || a != 0 {
		t.Errorf("inclusive: got rate=%v amount=%v, want 0/0 (VAT stays embedded)", r, a)
	}
	if r, a := documentVAT(s, ReceiptSettingsView{TaxMode: "none"}); r != 0 || a != 0 {
		t.Errorf("none: got rate=%v amount=%v, want 0/0", r, a)
	}
}

// The sale must render through the SAME templates the Documents menu uses,
// without error, for every supported form.
func TestRenderSaleThroughDocumentTemplates(t *testing.T) {
	s := sampleSaleForDoc()
	store := dochtml.StoreInfo{Name: s.StoreName, Address: s.StoreAddress, Phone: s.StorePhone, TaxID: s.StoreTaxID}

	for docType := range saleDocumentTypes {
		html, err := dochtml.RenderDocumentHTML(buildSaleDocData(s, docType, 7, 63), store)
		if err != nil {
			t.Errorf("render %s: unexpected error %v", docType, err)
			continue
		}
		if !strings.Contains(html, s.SaleNumber) {
			t.Errorf("render %s: HTML missing document number %q", docType, s.SaleNumber)
		}
	}
}
