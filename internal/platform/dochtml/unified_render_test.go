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

func TestProfile_PaymentSectionOnlyForInvoiceAndBill(t *testing.T) {
	for _, docType := range []string{"INVOICE", "BILL"} {
		if !profileFor(docType).ShowPayBox {
			t.Fatalf("%s should show payment section", docType)
		}
	}
	for _, docType := range []string{"QUOTATION", "DELIVERY_ORDER", "RECEIPT", "TAX_INVOICE", "CREDIT_NOTE"} {
		if profileFor(docType).ShowPayBox {
			t.Fatalf("%s should not show payment section", docType)
		}
	}
}

func TestUnifiedRender_PaymentBoxUsesStoreBankAccounts(t *testing.T) {
	doc := makeDoc("INVOICE", 1)
	html, err := RenderUnifiedDocumentHTML(doc, StoreInfo{
		Name: "ร้าน",
		BankAccounts: []BankAccountInfo{{
			BankName:    "ธนาคารทดสอบ",
			AccountNo:   "123-4-56789-0",
			AccountName: "ร้านทดสอบ",
		}},
	})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	for _, want := range []string{"ธนาคารทดสอบ", "เลขบัญชี 123-4-56789-0", "ร้านทดสอบ"} {
		if !strings.Contains(html, want) {
			t.Fatalf("payment box missing bank setting %q", want)
		}
	}
}
func TestUnifiedRender_PaymentBoxRendersOnlyOneBankAccount(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("INVOICE", 1), StoreInfo{
		Name: "ร้าน",
		BankAccounts: []BankAccountInfo{
			{BankName: "ธนาคารกรุงเทพ", AccountNo: "5664332667", AccountName: "บัญชีหนึ่ง"},
			{BankName: "ธนาคารกรุงไทย", AccountNo: "32142495834", AccountName: "บัญชีสอง"},
		},
	})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !strings.Contains(html, "ธนาคารกรุงเทพ") || !strings.Contains(html, "5664332667") {
		t.Fatal("the selected first bank account was not rendered")
	}
	if strings.Contains(html, "ธนาคารกรุงไทย") || strings.Contains(html, "32142495834") {
		t.Fatal("payment box must render exactly one bank account")
	}
	if !strings.Contains(html, `.bank-sub{ display:flex; align-items:baseline; gap:1.5mm; margin:-0.5mm 0 1.5mm 3.5mm; font-size:10px; line-height:1.2; font-weight:700; color:var(--ink); }`) {
		t.Fatal("bank detail row must match the payment box font scale and use bold text")
	}
}

func TestUnifiedRender_NonPaymentFooterReleasesPaymentBoxSpace(t *testing.T) {
	paymentFooter := profileFor("INVOICE").footerBlockH(false)
	nonPaymentFooter := profileFor("QUOTATION").footerBlockH(false)
	if paymentFooter-nonPaymentFooter != hPayBox-hRemarks {
		t.Fatalf("non-payment footer should release paybox space: payment=%v non-payment=%v", paymentFooter, nonPaymentFooter)
	}
}

func TestUnifiedRender_NonPaymentDocumentsUseRemarksWithoutPaymentBox(t *testing.T) {
	for _, docType := range []string{"QUOTATION", "DELIVERY_ORDER", "RECEIPT", "TAX_INVOICE", "CREDIT_NOTE"} {
		html, err := RenderUnifiedDocumentHTML(makeDoc(docType, 1), StoreInfo{Name: "ร้าน"})
		if err != nil {
			t.Fatalf("%s: execute error: %v", docType, err)
		}
		if strings.Contains(html, `class="paybox"`) {
			t.Fatalf("%s must not render payment box", docType)
		}
		if !strings.Contains(html, `class="remarks remarks-inline"`) {
			t.Fatalf("%s must render full-width remarks beside the summary", docType)
		}
	}
}

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

func TestUnifiedRender_DeliveryOrderIncludesPOReferenceRow(t *testing.T) {
	doc := makeDoc("DELIVERY_ORDER", 1)
	doc.PORefNo = "PO256909-0015"

	html, err := RenderUnifiedDocumentHTML(doc, StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	want := `<tr><td>อ้างอิงใบสั่งซื้อ (Ref. PO)</td><td class="b">PO256909-0015</td></tr>`
	if !strings.Contains(html, want) {
		t.Fatalf("missing PO reference row %q", want)
	}
}

func TestUnifiedRender_BillUsesDeliveryOrderRegister(t *testing.T) {
	doc := makeDoc("BILL", 0)
	due := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	doc.BillRows = []BillRow{{DocumentNo: "DO256909-0001", IssueDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), DueDate: &due, Amount: 1250}}

	html, err := RenderUnifiedDocumentHTML(doc, StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	for _, want := range []string{"ใบส่งสินค้า (DO)", "วันที่ออกเอกสาร", "วันครบกำหนด", "จำนวนเงิน", "DO256909-0001", "1,250.00"} {
		if !strings.Contains(html, want) {
			t.Fatalf("billing notice missing %q", want)
		}
	}
	if strings.Contains(html, `<table class="items">`) {
		t.Fatalf("billing notice must not render product-line table")
	}
	if !strings.Contains(html, `<tr class="filler"><td>&nbsp;</td><td class="left"></td><td></td><td></td><td class="num"></td></tr>`) {
		t.Fatal("billing notice must fill the remaining table rows")
	}
}

func TestUnifiedRender_BillEmptyRowsKeepTableCells(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("BILL", 0), StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if strings.Contains(html, "ไม่มีใบส่งสินค้าที่ค้างวางบิล") {
		t.Fatal("empty BILL must not replace the table row with a message")
	}
	if !strings.Contains(html, `<tr class="filler"><td>&nbsp;</td><td class="left"></td><td></td><td></td><td class="num"></td></tr>`) {
		t.Fatal("empty BILL must render five empty table cells")
	}
	if !strings.Contains(html, `.bill-items td{ height:var(--row-h); border:1px solid var(--line); padding:0.8mm 2mm; vertical-align:middle; text-align:center; }`) {
		t.Fatal("BILL rows must use the same row height and cell padding as other document tables")
	}
}
func TestUnifiedRender_CopySeparation(t *testing.T) {
	html, err := RenderUnifiedDocumentCopies(makeDoc("TAX_INVOICE", 3), StoreInfo{Name: "ร้านทดสอบ"}, -1)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if got := strings.Count(html, `class="copy-break"`); got < 2 {
		t.Fatalf("copy set should wrap each copy in .copy-break (>=2), got %d", got)
	}
	// screen separator must be present so the preview drawer splits the copies
	if !strings.Contains(html, "copy-break:not(:last-child)") {
		t.Fatalf("missing on-screen separator between copies")
	}
}
