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
		DocumentNoFull:  "INV256909-0001",
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

func TestUnifiedRender_QuotationIntroUsesSummaryBelowDocumentInfo(t *testing.T) {
	doc := makeDoc("QUOTATION", 1)
	doc.QuotationSummary = "เครื่องใช้สำนักงานและอุปกรณ์ที่เกี่ยวข้อง"
	html, err := RenderUnifiedDocumentHTML(doc, StoreInfo{Name: "ร้านทดสอบ"})
	if err != nil {
		t.Fatalf("render quotation: %v", err)
	}
	infoEnd := strings.Index(html, `class="doc-info-row"`)
	introStart := strings.Index(html, `class="quotation-intro"`)
	if infoEnd < 0 || introStart < 0 || introStart <= infoEnd {
		t.Fatal("quotation intro should be directly below doc-info-row")
	}
	if !strings.Contains(html, "<strong>ร้านทดสอบ</strong> ขอเสนอราคา <strong>เครื่องใช้สำนักงานและอุปกรณ์ที่เกี่ยวข้อง</strong> เพื่อโปรดพิจารณา โดยมีรายละเอียดดังต่อไปนี้") {
		t.Fatal("quotation intro text missing")
	}
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

func TestUnifiedRender_DocumentMetaAndPartyShareSixtyFortyRow(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("DELIVERY_ORDER", 1), StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	for _, want := range []string{
		`class="doc-id">INV256909-0001</div>`,
		`class="doc-info-row"`,
		`combined-parties"`,
		`class="doc-meta-panel"`,
		`.parties{ display:flex; flex-direction:column; gap:0; margin:0; border:1px solid var(--line); border-right:0; min-width:0; }`,
		`.combined-parties .section-body{ gap:.75mm; }`,
		`class="doc-meta-line"`,
		`class="header-section-title"`,
		`class="info-row"`,
		`รายละเอียดเอกสาร · DOCUMENT DETAILS`,
		`ชื่อหน่วยงาน`,
		`เลขผู้เสียภาษี`,
		`ข้อมูลลูกค้าและจัดส่ง · CUSTOMER / DELIVERY`,
		`วันที่จัดส่ง`,
		`.doc-info-row .parties{ flex:6 1 0; min-width:0; margin:0; }`,
		`.doc-meta-panel{ flex:4 1 0; min-width:0; display:flex; align-items:stretch; border:1px solid var(--line); padding:0 3mm 2mm; }`,
		`.header-section-title{ font-size:10px; font-weight:700; color:var(--ink); background:#f6f6f6; margin:0 -3mm 2mm; padding:1mm 3mm; line-height:1.5; border-bottom:1px solid var(--line); }`,
		`.info-row{ display:flex; align-items:flex-start; gap:2mm; min-width:0; line-height:1.5; }`,
		`.combined-parties .info-row{ line-height:1.35; }`,
		`.baht-text-row{ display:flex; gap:2mm; margin:0; padding:1.5mm 3mm; border:1px solid var(--line); border-top:0; background:#f6f6f6; font-size:13px; line-height:1.5; break-before:avoid; page-break-before:avoid; }`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("document info layout missing %q", want)
		}
	}
	if strings.Contains(html, `<table class="doc-meta">`) {
		t.Fatal("doc-meta should render as plain text, not a table box")
	}
	for _, removed := range []string{
		`ข้อมูลลูกค้าและเอกสาร (Customer &amp; Document Information)`,
		`class="doc-purpose"`,
		`class="doc-title-en"`,
	} {
		if strings.Contains(html, removed) {
			t.Fatalf("removed document header content still rendered: %q", removed)
		}
	}
}

func TestUnifiedRender_NonPaymentFooterReleasesPaymentBoxSpace(t *testing.T) {
	paymentFooter := profileFor("INVOICE").footerBlockH(false)
	nonPaymentFooter := profileFor("QUOTATION").footerBlockH(false)
	if paymentFooter-nonPaymentFooter != hPayBox-hRemarks-hTerms {
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
		if !strings.Contains(html, `class="foot-grid remarks-grid"`) {
			t.Fatalf("%s footer must use the stretch layout for remarks and summary", docType)
		}
		if !strings.Contains(html, `.remarks-inline{ flex:1; min-width:0; height:100%; align-self:stretch; margin-top:0; }`) {
			t.Fatalf("%s remarks must keep full-height styling", docType)
		}
		if !strings.Contains(html, `.foot-grid.remarks-grid{ align-items:stretch; }`) {
			t.Fatalf("%s remarks must stretch to the summary height", docType)
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
func TestUnifiedRender_InvoiceUsesDedicatedFooterPageWhenNeeded(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("INVOICE", 18), StoreInfo{Name: "ร้านทดสอบ"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !strings.Contains(html, "หน้า 2 / 2") {
		t.Fatalf("expected exactly 2 pages")
	}
	if !strings.Contains(html, "footer-only") {
		t.Fatalf("invoice must render a dedicated footer page")
	}
	if n := strings.Count(html, `<table class="items">`); n != 1 {
		t.Fatalf("หน้าสินค้าควรมีตารางเดียว แต่พบ %d ตาราง", n)
	}
	if strings.Contains(html, `class="filler"`) {
		t.Fatalf("หน้าสินค้าที่มีหน้าถัดไปไม่ควรมี filler row")
	}
	if !strings.Contains(html, "Grand Total") {
		t.Fatalf("footer (สรุปยอด) ต้อง render บนหน้า footer")
	}
}

func TestUnifiedRender_AllDocumentTypesMultiPageWithoutFooterOnly(t *testing.T) {
	wantPages := map[string]int{
		"QUOTATION": 2, "INVOICE": 3, "BILL": 3,
		"TAX_INVOICE": 2, "RECEIPT": 2, "CREDIT_NOTE": 2,
		"DELIVERY_ORDER": 3,
	}
	for docType, want := range wantPages {
		html, err := RenderUnifiedDocumentHTML(makeDoc(docType, 40), StoreInfo{Name: "ร้านทดสอบ"})
		if err != nil {
			t.Fatalf("%s: execute error: %v", docType, err)
		}
		if got := strings.Count(html, `<div class="page`) - strings.Count(html, `<div class="page-no`); got != want {
			t.Fatalf("%s: got %d logical pages, want %d", docType, got, want)
		}
		if docType == "QUOTATION" || docType == "TAX_INVOICE" || docType == "RECEIPT" || docType == "CREDIT_NOTE" {
			if strings.Contains(html, "footer-only") {
				t.Fatalf("%s: footer should fit with final product page", docType)
			}
		} else if !strings.Contains(html, "footer-only") {
			t.Fatalf("%s: expected dedicated footer page", docType)
		}
	}
}
func TestUnifiedRender_MultiPageDoesNotPadNonFinalPage(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("QUOTATION", 40), StoreInfo{Name: "ร้านทดสอบ"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	firstPage := strings.Index(html, `<div class="page`)
	secondPage := strings.Index(html[firstPage+1:], `<div class="page`)
	if firstPage < 0 || secondPage < 0 {
		t.Fatal("expected a multi-page quotation")
	}
	secondPage += firstPage + 1
	if strings.Contains(html[firstPage:secondPage], `class="filler"`) {
		t.Fatal("first page must contain product rows only, without filler rows")
	}
	if strings.Contains(html, `class="filler"`) {
		t.Fatal("multi-page document must not add filler rows that can push the footer to a new page")
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
	if !strings.Contains(html, `.page-no{ position:absolute; top:6mm; right:12mm; bottom:auto; font-size:9px; color:var(--muted); white-space:nowrap; }`) {
		t.Fatal("page number must be absolutely positioned at the top without consuming page flow")
	}
}

func TestUnifiedRender_DeliveryOrderIncludesPOReferenceRow(t *testing.T) {
	doc := makeDoc("DELIVERY_ORDER", 1)
	doc.PORefNo = "PO256909-0015"

	html, err := RenderUnifiedDocumentHTML(doc, StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	want := `<div class="doc-meta-line"><span class="doc-meta-label">อ้างอิงใบสั่งซื้อ</span><span class="doc-meta-value">PO256909-0015</span></div>`
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
	if !strings.Contains(html, `.page:last-child{ page-break-after:auto; margin-bottom:0; }`) {
		t.Fatalf("last page in each copy must not create a trailing blank PDF page")
	}
	if !strings.Contains(html, `@media print{ body{background:#fff;} .page{ margin:0; box-shadow:none; } .copy-break{ display:contents; page-break-after:auto !important; } }`) {
		t.Fatalf("print copy wrapper must not introduce an extra blank page")
	}
}
