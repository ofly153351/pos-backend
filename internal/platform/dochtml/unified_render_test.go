package dochtml

import (
	"strings"
	"testing"
	"time"
)

// makeDoc builds a DocData with n line items for render smoke tests.
func makeDoc(docType string, n int) DocData {
	items := make([]DocItem, n)
	for i := range items {
		items[i] = DocItem{
			SKU:           "SKU-" + strings.Repeat("0", 0),
			Description:   "สินค้าทดสอบ",
			DescriptionEn: "Test product",
			Unit:          "ชิ้น",
			Quantity:      float64(i + 1),
			UnitPrice:     100,
			DiscountValue: 5,
			Amount:        float64(i+1) * 95,
		}
	}
	due := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	note := "ทดสอบหมายเหตุ"
	return DocData{
		Type:            docType,
		DocumentNoFull:  "DOC-0001",
		DocumentDate:    time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
		DueDate:         &due,
		ValidUntil:      &due,
		DeliveryDate:    &due,
		InvoiceRefNo:    "INV-9",
		CustomerName:    "ลูกค้า ทดสอบ",
		CustomerPhone:   "0812345678",
		CustomerAddress: "123 ถนนทดสอบ",
		DeliveryAddress: "456 คลังสินค้า",
		StaffName:       "พนักงาน",
		Items:           items,
		Subtotal:        1000,
		TotalDiscount:   50,
		VatRate:         7,
		VatAmount:       66.5,
		TotalAmount:     1016.5,
		PreVatAmount:    950,
		ShippingFee:     20,
		Notes:           &note,
	}
}

var renderTypes = []string{
	"DELIVERY_ORDER", "INVOICE", "RECEIPT", "TAX_INVOICE", "QUOTATION", "BILL", "CREDIT_NOTE",
}

// Every document type must execute the unified template without error and emit the
// page-number footer + grand-total label — catches template field-name typos that
// only surface at execution time.
func TestUnifiedRender_AllTypesExecute(t *testing.T) {
	store := StoreInfo{
		Name: "ร้านทดสอบ", Address: "ที่อยู่ร้าน", Phone: "021112222", TaxID: "0105500000001",
		BankAccounts: []BankAccountInfo{{BankName: "ธ.ทดสอบ", AccountNo: "123-4-56789-0", AccountName: "ร้านทดสอบ"}},
	}
	for _, dt := range renderTypes {
		html, err := RenderUnifiedDocumentHTML(makeDoc(dt, 3), store)
		if err != nil {
			t.Fatalf("%s: execute error: %v", dt, err)
		}
		// Page-number footer must render (proves the page loop + fields execute);
		// row budget varies per type so we don't assert an exact page count here.
		if !strings.Contains(html, "หน้า 1 / ") {
			t.Fatalf("%s: missing page-number footer marker", dt)
		}
		if !strings.Contains(html, "Grand Total") {
			t.Fatalf("%s: missing grand total label", dt)
		}
	}
}

// A long item list must paginate into multiple pages, repeat the table header, show
// a "Continued" marker on non-final pages, and carry the summary onto the last page.
func TestUnifiedRender_MultiPage(t *testing.T) {
	store := StoreInfo{Name: "ร้านทดสอบ"}
	html, err := RenderUnifiedDocumentHTML(makeDoc("DELIVERY_ORDER", 60), store)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !strings.Contains(html, "หน้า 1 / ") || strings.Contains(html, "หน้า 1 / 1") {
		t.Fatalf("expected multi-page output, got single page")
	}
	if !strings.Contains(html, "Grand Total") {
		t.Fatalf("summary missing on last page")
	}
	// Single-page docs pin the footer via the .single class; multi-page must NOT.
	if strings.Contains(html, "page single") {
		t.Fatalf("multi-page output must not carry the single-page footer-pin class")
	}
}

// เมื่อสินค้าลงหน้าต่อเนื่องได้หมดแต่ใส่ footer ไม่พอ (เช่น INVOICE 18 รายการ):
// หน้าสุดท้ายเป็น footer-only (ไม่มีตารางสินค้า) และหน้าสินค้าถูกเติม ledger ให้เต็ม
// (ไม่เหลือช่องว่างดิบท้ายหน้า).
func TestUnifiedRender_FooterOnlyLastPage(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("INVOICE", 18), StoreInfo{Name: "ร้านทดสอบ"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !strings.Contains(html, "หน้า 2 / 2") {
		t.Fatalf("expected exactly 2 pages")
	}
	if !strings.Contains(html, "footer-only") {
		t.Fatalf("missing footer-only page class")
	}
	// ตารางสินค้ามีแค่หน้าเดียว (หน้า footer-only ไม่มีตาราง)
	if n := strings.Count(html, `<table class="items">`); n != 1 {
		t.Fatalf("ตารางสินค้าควรมีแค่ 1 (หน้าสินค้า) แต่พบ %d — หน้า footer-only ไม่ควรมีตาราง", n)
	}
	// หน้าสินค้าถูกเติม ledger ให้เต็ม (18 รายการ < ความจุหน้า → ต้องมี filler)
	if !strings.Contains(html, `class="filler"`) {
		t.Fatalf("หน้าสินค้าควรถูกเติม ledger ให้เต็ม (filler) ไม่เหลือช่องว่างดิบ")
	}
	if !strings.Contains(html, "Grand Total") {
		t.Fatalf("footer (สรุปยอด) ต้อง render บนหน้า footer-only")
	}
}

// A single-item doc must be one page and carry the footer-pin class.
func TestUnifiedRender_SinglePagePin(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("RECEIPT", 1), StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !strings.Contains(html, "page single") {
		t.Fatalf("single-page doc must carry .single footer-pin class")
	}
	if !strings.Contains(html, "หน้า 1 / 1") {
		t.Fatalf("expected single page footer")
	}
}
