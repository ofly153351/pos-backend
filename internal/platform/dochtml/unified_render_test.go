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
		`.page{ width:auto; min-height:0; padding:0; margin:1rem; background:#fff; display:block; position:relative; }`,
		`body{ font-family:'Sarabun','Tahoma',sans-serif; color:var(--ink); font-size:11px; background:#fff; -webkit-print-color-adjust:exact; print-color-adjust:exact; }`,
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
		`.header-section-title{ font-size:10px; font-weight:700; color:var(--ink); background:#fff; margin:0 -3mm 2mm; padding:1mm 3mm; line-height:1.5; border-bottom:1px solid var(--line); }`,
		`.info-row{ display:flex; align-items:flex-start; gap:2mm; min-width:0; line-height:1.5; }`,
		`.combined-parties .info-row{ line-height:1.35; }`,
		`.baht-text-row{ display:flex; gap:2mm; margin:0; padding:1.5mm 3mm; border:1px solid var(--line); border-top:1px solid var(--line); background:#fff; font-size:13px; line-height:1.5; break-before:avoid; page-break-before:avoid; }`,
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

func TestUnifiedRender_NonPaymentFooterUsesNaturalFlow(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("QUOTATION", 1), StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("render quotation: %v", err)
	}
	if !strings.Contains(html, `.doc-footer{ margin-top:3mm; break-inside:avoid; page-break-inside:avoid; }`) {
		t.Fatal("footer must rely on natural flow with a non-splitting rule")
	}
	if strings.Contains(html, "footerBlockH") || strings.Contains(html, "rowsPerPage") {
		t.Fatal("renderer must not embed the removed server-side pagination geometry")
	}
}

func TestUnifiedRender_EndingBlockStaysInNaturalFlow(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("RECEIPT", 3), StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("render receipt: %v", err)
	}
	if !strings.Contains(html, `data-item-count="3"`) || !strings.Contains(html, `.doc-end{ margin-top:0; }`) {
		t.Fatal("document ending block must remain in normal flow")
	}
	for _, removed := range []string{
		`.page[data-item-count="1"],.page[data-item-count="2"],.page[data-item-count="3"],.page[data-item-count="4"]{ display:flex; flex-direction:column; min-height:calc(297mm - 12mm - 16mm); }`,
		`.page[data-item-count="1"] .doc-end,.page[data-item-count="2"] .doc-end,.page[data-item-count="3"] .doc-end,.page[data-item-count="4"] .doc-end{ margin-top:auto; }`,
	} {
		if strings.Contains(html, removed) {
			t.Fatalf("removed footer pinning rule still rendered: %q", removed)
		}
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
		if !strings.Contains(html, `class="remarks remarks-under-table"`) {
			t.Fatalf("%s must render remarks directly below the item table", docType)
		}
		if !strings.Contains(html, `class="remarks-summary-row"`) {
			t.Fatalf("%s must place remarks and summary in one flex row", docType)
		}
		if strings.Contains(html, `class="foot-grid remarks-grid"`) {
			t.Fatalf("%s footer must not use the old side-by-side remarks layout", docType)
		}
		if !strings.Contains(html, `.remarks-under-table{ margin-top:2mm; break-inside:avoid; page-break-inside:avoid; }`) {
			t.Fatalf("%s remarks must keep below-table print-safe styling", docType)
		}
		if !strings.Contains(html, "เงื่อนไขและหมายเหตุ") {
			t.Fatalf("%s remarks heading is missing", docType)
		}
	}
}

// The renderer emits one natural flow. Paged.js is responsible for physical pages.
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
		if strings.Contains(html, `class="running-header"`) || strings.Contains(html, "document-header") {
			t.Fatalf("%s: running header must not be rendered", dt)
		}
		if !strings.Contains(html, "Grand Total") {
			t.Fatalf("%s: missing grand total label", dt)
		}
	}
}

func TestUnifiedRender_NaturalFlowUsesPagedRules(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("DELIVERY_ORDER", 200), StoreInfo{Name: "ร้านทดสอบ"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if got := strings.Count(html, `<table class="items">`); got != 1 {
		t.Fatalf("expected one continuous product table, got %d", got)
	}
	if got := strings.Count(html, `<tr>`); got < 200 {
		t.Fatalf("expected all product rows in the natural flow, got %d row tags", got)
	}
	for _, marker := range []string{
		`thead{ display:table-header-group; }`,
		`tr{ break-inside:avoid; }`,
		`.doc-footer{ margin-top:3mm; break-inside:avoid; page-break-inside:avoid; }`,
		`.pagedjs_page{ position:relative; }`,
		`@media screen{`,
		`  body{ background:#e2e8f0; }`,
		`  .pagedjs_pages{`,
		`    display:flex;`,
		`    flex-direction:column;`,
		`    align-items:center;`,
		`    gap:16mm;`,
		`    padding:8mm;`,
		`    background:#e2e8f0;`,
		`  }`,
		`  .pagedjs_margin-bottom{ display:none !important; }`,
		`  .pagedjs_page + .pagedjs_page .pagedjs_bleed-top{`,
		`    display:block;`,
		`    min-height:2mm;`,
		`    background:#e2e8f0;`,
		`    border-top:2px dashed #94a3b8;`,
		`  }`,
		`  .pagedjs_page{ flex:0 0 auto; background:#fff; box-shadow:0 2px 8px rgba(15,23,42,.16); }`,
		`  .pagedjs_page:not(:last-child){ margin-bottom:16mm; }`,
		`}`,
		`.pos-page-number{ position:absolute; top:8mm; right:12mm; color:var(--muted); font-size:9px; white-space:nowrap; }`,
		`.screen-page-number{ position:absolute; top:8mm; right:12mm; z-index:2; color:var(--muted); font-size:9px; white-space:nowrap; }`,
		`<div class="screen-page-number">หน้า 1</div>`,
		`.items .desc{ display:block; overflow-wrap:anywhere; white-space:normal; line-height:1.2; }`,
		`.items tr{ break-inside:avoid; page-break-inside:avoid; }`,
	} {
		if !strings.Contains(html, marker) {
			t.Fatalf("missing Paged.js pagination rule %q", marker)
		}
	}
	if strings.Contains(html, `class="filler"`) {
		t.Fatal("natural multi-page flow must not use manual filler rows")
	}
	if strings.Contains(html, `class="footer-only"`) {
		t.Fatal("natural flow must not manufacture a footer-only page")
	}
}

func TestUnifiedRender_SinglePageDoesNotUseManualPagination(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("RECEIPT", 1), StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if got := strings.Count(html, `<div class="page`); got != 1 {
		t.Fatalf("expected one document flow, got %d", got)
	}
	if strings.Contains(html, "หน้า 1 / 1") || strings.Contains(html, `.page-no{`) {
		t.Fatal("page numbers must come from Paged.js counters, not manual markup")
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

func TestUnifiedRender_DeliveryOrderUsesInvoiceReferenceLabel(t *testing.T) {
	doc := makeDoc("DELIVERY_ORDER", 1)
	doc.InvoiceRefNo = "INV256909-0001"

	html, err := RenderUnifiedDocumentHTML(doc, StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	want := `<div class="doc-meta-line"><span class="doc-meta-label">อ้างอิงใบแจ้งหนี้</span><span class="doc-meta-value">INV256909-0001</span></div>`
	if !strings.Contains(html, want) {
		t.Fatalf("missing invoice reference row %q", want)
	}
	if strings.Contains(html, "อ้างอิงใบกำกับภาษี") {
		t.Fatal("delivery order must not label the invoice reference as a tax invoice")
	}
}

func TestUnifiedRender_CreditNoteUsesTaxInvoiceReferenceLabel(t *testing.T) {
	doc := makeDoc("CREDIT_NOTE", 1)
	doc.InvoiceRefNo = "TAX256909-0001"

	html, err := RenderUnifiedDocumentHTML(doc, StoreInfo{Name: "ร้าน"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	want := `<div class="doc-meta-line"><span class="doc-meta-label">อ้างอิงใบกำกับภาษี (Ref. Tax Invoice)</span><span class="doc-meta-value">TAX256909-0001</span></div>`
	if !strings.Contains(html, want) {
		t.Fatalf("missing tax invoice reference row %q", want)
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
	if !strings.Contains(html, `<tbody>`) {
		t.Fatal("billing notice must render a table body")
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
	if !strings.Contains(html, `.copy-break:last-child{ break-after:auto; page-break-after:auto; }`) {
		t.Fatalf("last copy must not create a trailing blank page")
	}
	if !strings.Contains(html, `data-copy-purpose="`) {
		t.Fatalf("copy flow must expose a copy marker for page numbering")
	}
	if !strings.Contains(html, `.copy-break{ break-after:page; page-break-after:always; }`) {
		t.Fatalf("copy flow must force a page break between copies")
	}
}

func TestUnifiedRender_ContinuationPagesHaveMinimalRunningHeader(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(makeDoc("INVOICE", 40), StoreInfo{Name: "ร้านทดสอบ"})
	if err != nil {
		t.Fatalf("render invoice: %v", err)
	}
	for _, want := range []string{
		`.hdr-mini{ display:none; }`,
		`.pos-continuation-header{ position:absolute; top:14mm; left:12mm; right:12mm;`,
		`<span class="m-store">ร้านทดสอบ</span>`,
		`<span class="m-title">ใบแจ้งหนี้ · INV256909-0001 (ต่อ / cont.)</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("continuation header markup missing %q", want)
		}
	}
}
