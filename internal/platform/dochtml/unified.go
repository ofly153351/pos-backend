package dochtml

// =============================================================================
//  ระบบเอกสารรวม (Unified Document Renderer) — drop-in สำหรับ render.go
// -----------------------------------------------------------------------------
//  แทนที่ 5 template แยก ด้วย template กลางตัวเดียว + paginator ตัวเดียว
//  ทุกชนิด (INVOICE/RECEIPT/TAX_INVOICE/QUOTATION/BILL/CREDIT_NOTE/DELIVERY_ORDER)
//  route ผ่าน RenderUnifiedDocumentHTML() → ความต่างตามชนิดอยู่ใน docProfile
//
//  แนวทาง: A (server arithmetic + fixed row height) — ไม่ใช่ client DOM measure
//    • backend รู้ TotalPages ก่อน render  → "หน้า X / Y" ฟรี
//    • ทุกแถวสูงเท่ากัน (CSS clamp 2 บรรทัด) → นับแถวแม่น
//    • footer ไม่ลอย = flexbox spacer + conditional class (ไม่ใช้ JS, ไม่มี race)
//
//  ─── ASSUMPTIONS (map ให้ตรง struct จริงของคุณก่อนใช้) ──────────────────────
//    • มี helper ระดับ package อยู่แล้ว:  money(float64) string, thaiDate(time.Time) string
//      → อย่า redefine ในไฟล์นี้
//    • DocData field ที่อ้างถึง: DocumentNoFull, DocumentDate(time.Time),
//      DueDate/ValidUntil(*time.Time), CustomerName/CustomerAddress/CustomerPhone/
//      CustomerTaxID(string), StaffName, SalespersonName, InvoiceRefNo(string),
//      ShippingFee/PreVatAmount/Subtotal/TotalDiscount/VatRate/VatAmount/TotalAmount(float64),
//      DeliveryAddress/DeliveryContact/DeliveryPhone(string),
//      DeliveryDate(*time.Time), Notes(*string), QRPaymentURL(template.URL), Items([]DocItem)
//    • DocItem: SKU, Description, DescriptionEn, Unit(string), Quantity(float64),
//      UnitPrice/DiscountValue/Amount(float64)
//    • StoreInfo: Name/Address/Phone/TaxID/Branch/LogoURL(string),
//      BankAccounts[]{BankName,AccountNo,AccountName}
//    ↳ ถ้าชื่อ/ชนิด field ต่าง แก้เฉพาะ builder ด้านล่าง (ไม่ต้องแตะ template/paginator)
// =============================================================================

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"

	"pos-backend/internal/platform/doccopy"
)

// NOTE: GEOMETRY (§1) + PAGINATOR (§2) ย้ายไปไฟล์ pagination.go แล้ว
// (geometry/rowsPerPage/buildGeometry/pageSlice/paginate/packRows + consts)
// อยู่ package เดียวกัน — BuildDocumentView ด้านล่างเรียกใช้ได้ตามปกติ

// =============================================================================
//  3) DOC PROFILE — ความต่างตามชนิดเอกสาร (ตาราง §10.1)
// =============================================================================

type docProfile struct {
	TitleTH, TitleEN                             string
	Badge                                        string // "" = ไม่โชว์
	ShowDiscount                                 bool   // คอลัมน์ส่วนลด
	ShowPayBox                                   bool   // กล่องชำระเงิน + QR
	ShowDeliveryBox                              bool   // กล่อง "ที่อยู่จัดส่ง" (เฉพาะใบส่งของ)
	IsDelivery                                   bool   // มี ค่าจัดส่ง + ยอดก่อน VAT ใน summary
	PayCash                                      bool   // ติ๊ก "เงินสด" อัตโนมัติ (ใบเสร็จ)
	SpecialLabel                                 string // ป้ายฟิลด์พิเศษหัวขวา ("" = ไม่มี)
	ShowTerms                                    bool   // quotation terms block below the footer grid
	SigLeftTH, SigLeftEN, SigRightTH, SigRightEN string
}

// footerBlockH = ความสูงรวมของก้อนท้าย (summary + paybox + remarks + sig) ตาม profile/data
func (p docProfile) footerBlockH(hasNotes bool) float64 {
	rows := 3.0 // subtotal, VAT, grand total
	if p.ShowDiscount {
		rows++
	}
	if p.IsDelivery {
		rows += 2 // ค่าจัดส่ง + ยอดก่อน VAT
	}
	summaryH := rows*hSumRow + hSumExtra

	// .foot-grid lays the pay box and the summary SIDE BY SIDE (flexbox row), so the
	// grid's height is the taller of the two — not their sum. Reserving the sum here
	// double-counts the pay box and starves every page of rows.
	gridH := summaryH
	if p.ShowPayBox && hPayBox > gridH {
		gridH = hPayBox
	}
	if !p.ShowPayBox && hRemarks > gridH {
		// Non-payment documents place remarks beside the summary inside foot-grid.
		// It must not be reserved again as a stacked block below the grid.
		gridH = hRemarks
	}

	h := gridH
	// กล่องหมายเหตุขึ้นเสมอ (เป็น form field สำหรับเขียน ไม่ผูกกับว่ามี Notes ไหม)
	// เอกสารที่มี paybox วาง remarks เป็นกล่องแยกด้านล่าง; เอกสารอื่นวางไว้ใน grid แล้ว
	_ = hasNotes
	if p.ShowPayBox {
		h += hRemarks
	}
	if p.IsDelivery {
		h += 8 // baht-text row below the item table
	}
	if p.ShowTerms {
		h += hTerms
	}
	h += hSigBlock // signatures stacked below remarks
	return h
}

func profileFor(docType string) docProfile {
	switch docType {
	case "DELIVERY_ORDER":
		return docProfile{
			TitleTH: "ใบส่งของ / ใบกำกับภาษี", TitleEN: "Delivery Note / Tax Invoice",
			Badge: "ต้นฉบับ (ORIGINAL)", ShowDiscount: true, ShowPayBox: false,
			ShowDeliveryBox: true, IsDelivery: true, SpecialLabel: "วันที่จัดส่ง (Delivery)",
			SigLeftTH: "ผู้ส่งสินค้า", SigLeftEN: "Delivered By", SigRightTH: "ผู้รับสินค้า", SigRightEN: "Received By",
		}
	case "INVOICE":
		return docProfile{
			TitleTH: "ใบแจ้งหนี้", TitleEN: "Invoice",
			ShowDiscount: true, ShowPayBox: true, SpecialLabel: "ครบกำหนด (Due)",
			SigLeftTH: "ผู้ออกเอกสาร", SigLeftEN: "Issued By", SigRightTH: "ผู้รับเอกสาร", SigRightEN: "Document Receiver",
		}
	case "RECEIPT":
		return docProfile{
			TitleTH: "ใบเสร็จรับเงิน", TitleEN: "Receipt",
			ShowDiscount: false, ShowPayBox: false, PayCash: true, SpecialLabel: "วันที่รับเงิน (Paid)",
			SigLeftTH: "ผู้รับเงิน", SigLeftEN: "Received By", SigRightTH: "ผู้จ่ายเงิน", SigRightEN: "Paid By",
		}
	case "TAX_INVOICE":
		return docProfile{
			TitleTH: "ใบกำกับภาษี", TitleEN: "Tax Invoice",
			Badge: "ต้นฉบับ (ORIGINAL)", ShowDiscount: true, ShowPayBox: false, SpecialLabel: "",
			SigLeftTH: "ผู้ออกเอกสาร", SigLeftEN: "Issued By", SigRightTH: "ผู้รับสินค้า", SigRightEN: "Goods Receiver",
		}
	case "QUOTATION":
		return docProfile{
			TitleTH: "ใบเสนอราคา", TitleEN: "Quotation",
			ShowDiscount: true, ShowPayBox: false, SpecialLabel: "", ShowTerms: true,
			SigLeftTH: "ผู้จัดทำ", SigLeftEN: "Prepared By", SigRightTH: "ผู้อนุมัติ", SigRightEN: "Approved By",
		}
	case "BILL":
		return docProfile{
			TitleTH: "ใบวางบิล", TitleEN: "Billing Notice",
			ShowDiscount: true, ShowPayBox: true, SpecialLabel: "ครบกำหนด (Due)",
			SigLeftTH: "ผู้ออกเอกสาร", SigLeftEN: "Issued By", SigRightTH: "ผู้รับวางบิล", SigRightEN: "Billing Receiver",
		}
	case "CREDIT_NOTE":
		return docProfile{
			TitleTH: "ใบลดหนี้", TitleEN: "Credit Note",
			ShowDiscount: false, ShowPayBox: false, SpecialLabel: "อ้างอิงใบกำกับภาษี (Ref. Tax Invoice)",
			SigLeftTH: "ผู้ออกเอกสาร", SigLeftEN: "Issued By", SigRightTH: "ผู้รับเอกสาร", SigRightEN: "Document Receiver",
		}
	default:
		return profileFor("INVOICE")
	}
}

// =============================================================================
//  4) VIEW MODEL — flatten ทุกอย่างเป็น string/bool ก่อนเข้า template
//     (template ไม่ยุ่งกับ pointer/time/float → ปลอดภัย + ไม่ต้องมี template func)
// =============================================================================

type itemView struct {
	No                       int
	SKU, Desc, DescEn, Unit  string
	Qty, UnitPrice, Discount string
	Amount                   string
	HasDiscount              bool
}

type pageView struct {
	Items              []itemView
	FillerRows         []struct{} // empty rows padding the grid to the page's row capacity
	PageNo, TotalPages int
	IsFirst, IsLast    bool
	FooterOnly         bool // หน้าสุดท้ายที่มีแต่ footer/สรุปยอด (ไม่มีตารางสินค้า)
}

type summaryLine struct {
	Label, Value string
	Strong       bool
}

type bankView struct{ Name, No, Holder string }

type billRowView struct {
	No                                     int
	DocumentNo, IssueDate, DueDate, Amount string
}

type renderView struct {
	// ร้าน
	StoreName, StoreAddr, StoreTaxID, StorePhone, StoreBranch string
	// LogoURL is template.URL so a base64 data: URI (embedded logo) is not stripped
	// to #ZgotmplZ by html/template's URL sanitizer.
	LogoURL template.URL
	// หัวเอกสาร
	TitleTH, TitleEN, Badge                       string
	Purpose                                       string // copy purpose tag e.g. "(สำหรับลูกค้า)"
	ShowSignature                                 bool   // render the signature block on this copy
	DocNo, DocDate                                string
	PriceValidity, PaymentTerms, DeliveryLeadTime string
	SpecialLabel, SpecialValue                    string
	// ลูกค้า / จัดส่ง
	CustomerName, CustomerTaxID, CustomerAddr, CustomerPhone           string
	QuotationSummary                                                   string
	StaffName, SalespersonName, RefNo, RefLabel, PORefNo, DeliveryDate string
	// QuotationRefNo = แถว "อ้างอิงใบเสนอราคา (Ref. Quotation)" ในตารางหัวเอกสาร
	QuotationRefNo                               string
	PriceTerms, DeliveryTerms                    string
	DeliveryAddr, DeliveryContact, DeliveryPhone string
	BahtText                                     string
	// flags
	ShowDiscount, ShowDeliveryBox, ShowPayBox, PayCash bool
	// footer
	Summary                                      []summaryLine
	Notes                                        string
	Banks                                        []bankView
	QRURL                                        template.URL
	BillRows                                     []billRowView
	SigLeftTH, SigLeftEN, SigRightTH, SigRightEN string
	// pages
	Pages        []pageView
	IsSinglePage bool
}

// ---- helpers (เฉพาะไฟล์นี้) -------------------------------------------------

func fmtQty(q float64) string {
	if q == float64(int64(q)) {
		return strconv.FormatInt(int64(q), 10)
	}
	return strconv.FormatFloat(q, 'f', -1, 64)
}

func specialValue(d DocData, docType string) string {
	switch docType {
	case "QUOTATION":
		if d.ValidUntil != nil {
			return thaiDate(*d.ValidUntil)
		}
	case "INVOICE", "BILL":
		if d.DueDate != nil {
			return thaiDate(*d.DueDate)
		}
	case "RECEIPT":
		return thaiDate(d.DocumentDate)
	case "CREDIT_NOTE":
		return d.InvoiceRefNo
	case "DELIVERY_ORDER":
		if d.DeliveryDate != nil {
			return thaiDate(*d.DeliveryDate)
		}
		return "_"
	}
	return ""
}

func termDays(days *int) string {
	if days == nil || *days <= 0 {
		return "_"
	}
	return strconv.Itoa(*days) + " วัน"
}

func quotationTerms(d DocData) (priceTerms, deliveryTerms string) {
	if d.Type != "QUOTATION" {
		return "", ""
	}
	validDays := "_"
	if d.PriceValidityDays != nil && *d.PriceValidityDays > 0 {
		validDays = strconv.Itoa(*d.PriceValidityDays)
	}
	deliveryDays := "_"
	if d.DeliveryLeadTimeDays != nil && *d.DeliveryLeadTimeDays > 0 {
		deliveryDays = strconv.Itoa(*d.DeliveryLeadTimeDays)
	}
	poDate := "_/_/_"
	if d.POReceivedDate != nil && !d.POReceivedDate.IsZero() {
		poDate = thaiDate(*d.POReceivedDate)
	}
	priceTerms = fmt.Sprintf("ราคานี้ยืนราคาเป็นระยะเวลา %s วัน นับจากวันที่ออกใบเสนอราคา", validDays)
	deliveryTerms = fmt.Sprintf("กำหนดส่งสินค้าภายใน %s วัน หลังจากได้รับใบสั่งซื้อวันที่ %s", deliveryDays, poDate)
	return priceTerms, deliveryTerms
}

func buildSummary(d DocData, p docProfile) []summaryLine {
	s := []summaryLine{
		{Label: "รวมมูลค่าสินค้า (Subtotal)", Value: money(d.Subtotal)},
	}
	if p.ShowDiscount && d.TotalDiscount > 0 {
		s = append(s, summaryLine{Label: "ส่วนลด (Discount)", Value: "- " + money(d.TotalDiscount)})
	}
	if p.IsDelivery && d.ShippingFee > 0 {
		s = append(s, summaryLine{Label: "ค่าจัดส่ง (Shipping)", Value: money(d.ShippingFee)})
	}
	if p.IsDelivery {
		s = append(s, summaryLine{Label: "ยอดก่อนภาษี (Pre-VAT)", Value: money(d.PreVatAmount)})
	}
	if d.VatRate > 0 {
		s = append(s, summaryLine{
			Label: fmt.Sprintf("ภาษีมูลค่าเพิ่ม %g%% (VAT %g%%)", d.VatRate, d.VatRate),
			Value: money(d.VatAmount),
		})
	}
	s = append(s, summaryLine{Label: "รวมทั้งสิ้น (Grand Total)", Value: money(d.TotalAmount), Strong: true})
	return s
}

// BuildDocumentView = mapping layer: DocData/StoreInfo → renderView (paginated)
func BuildDocumentView(d DocData, store StoreInfo) renderView {
	p := profileFor(d.Type)
	hasNotes := derefStr(d.Notes) != ""
	g := buildGeometry(p, hasNotes)

	slices := paginate(g, len(d.Items))
	totalPages := len(slices)

	// ความจุเชิงกายภาพต่อชนิดหน้า — ใช้ pad แถวเปล่า (ledger) ให้ตารางเต็มถึงล่าง/footer
	// ทุกหน้าที่มีตาราง (รวมหน้าต่อเนื่อง) เพื่อไม่ให้เหลือ "ช่องว่างดิบ" ท้ายหน้า
	capFL := g.rowsPerPage(g.headerFull, true)

	pages := make([]pageView, 0, totalPages)
	for idx, sl := range slices {
		items := make([]itemView, 0, sl.End-sl.Start)
		for j := sl.Start; j < sl.End; j++ {
			it := d.Items[j]
			hasDisc := p.ShowDiscount && it.DiscountValue > 0
			disc := ""
			if hasDisc {
				disc = money(it.DiscountValue)
			}
			items = append(items, itemView{
				No:          j + 1,
				SKU:         it.SKU,
				Desc:        it.Description,
				DescEn:      it.DescriptionEn,
				Unit:        it.Unit,
				Qty:         fmtQty(it.Quantity),
				UnitPrice:   money(it.UnitPrice),
				Discount:    disc,
				Amount:      money(it.Amount),
				HasDiscount: hasDisc,
			})
		}

		// หน้าสุดท้ายที่ไม่มีสินค้า = footer-only (สินค้าเต็มอยู่หน้าก่อนหน้าแล้ว)
		footerOnly := sl.IsLast && !sl.IsFirst && len(items) == 0

		// เติม filler เฉพาะเอกสารหน้าเดียวเท่านั้น
		// หน้าสุดท้ายของเอกสารหลายหน้าต้องเก็บพื้นที่ไว้สำหรับ footer/signature
		// หากเติม filler จะดัน footer ไปหน้าใหม่และสร้างหน้าว่าง/หน้าล้นใน PDF.
		fillerN := 0
		if sl.IsLast && sl.IsFirst {
			pageCap := capFL
			if fillerN = pageCap - len(items); fillerN < 0 {
				fillerN = 0
			}
		}

		pages = append(pages, pageView{
			Items: items, FillerRows: make([]struct{}, fillerN),
			PageNo: idx + 1, TotalPages: totalPages,
			IsFirst: sl.IsFirst, IsLast: sl.IsLast, FooterOnly: footerOnly,
		})
	}

	banks := make([]bankView, 0, 1)
	if p.ShowPayBox && len(store.BankAccounts) > 0 {
		// Store settings may contain several accounts, but a document must render
		// exactly one account. Selection is resolved by the document service before
		// reaching the renderer; keeping this boundary defensive prevents accidental
		// multi-account output if an older caller still passes the full list.
		b := store.BankAccounts[0]
		banks = append(banks, bankView{Name: b.BankName, No: b.AccountNo, Holder: b.AccountName})
	}

	priceTerms, deliveryTerms := quotationTerms(d)
	priceValidity := "_"
	if d.PriceValidityDays != nil && *d.PriceValidityDays > 0 {
		priceValidity = strconv.Itoa(*d.PriceValidityDays) + " วัน"
	}
	paymentTerms := "_"
	if d.CreditTermDays > 0 {
		paymentTerms = strconv.Itoa(d.CreditTermDays) + " วัน"
	}
	deliveryLeadTime := termDays(d.DeliveryLeadTimeDays)
	if d.Type != "QUOTATION" {
		priceValidity = ""
		paymentTerms = ""
		deliveryLeadTime = ""
	}
	billRows := make([]billRowView, 0, len(d.BillRows))
	for _, row := range d.BillRows {
		due := "_"
		if row.DueDate != nil {
			due = thaiDate(*row.DueDate)
		}
		billRows = append(billRows, billRowView{
			No:         len(billRows) + 1,
			DocumentNo: row.DocumentNo,
			IssueDate:  thaiDate(row.IssueDate),
			DueDate:    due,
			Amount:     money(row.Amount),
		})
	}
	return renderView{
		StoreName: store.Name, StoreAddr: store.Address, StoreTaxID: store.TaxID,
		StorePhone: store.Phone, StoreBranch: store.Branch, LogoURL: template.URL(store.LogoURL),

		TitleTH: p.TitleTH, TitleEN: p.TitleEN, Badge: p.Badge,
		// Single-render default: signatures shown (copy renderer overrides per variant).
		ShowSignature: true,
		DocNo:         d.DocumentNoFull, DocDate: thaiDate(d.DocumentDate),
		PriceValidity: priceValidity, PaymentTerms: paymentTerms, DeliveryLeadTime: deliveryLeadTime,
		SpecialLabel: p.SpecialLabel, SpecialValue: specialValue(d, d.Type),

		CustomerName: d.CustomerName, CustomerTaxID: derefStr(d.CustomerTaxID),
		CustomerAddr: d.CustomerAddress, CustomerPhone: d.CustomerPhone,
		QuotationSummary: strings.TrimSpace(d.QuotationSummary),
		StaffName:        d.StaffName, SalespersonName: d.SalespersonName, RefNo: d.InvoiceRefNo,
		RefLabel: func() string {
			if d.Type == "DELIVERY_ORDER" {
				return "อ้างอิงใบแจ้งหนี้"
			}
			return "อ้างอิงใบกำกับภาษี"
		}(),
		PORefNo: d.PORefNo, QuotationRefNo: d.QuotationRefNo,
		DeliveryDate: func() string {
			if d.DeliveryDate != nil {
				return thaiDate(*d.DeliveryDate)
			}
			return "_"
		}(),
		PriceTerms: priceTerms, DeliveryTerms: deliveryTerms,
		DeliveryAddr:    d.DeliveryAddress,
		DeliveryContact: d.DeliveryContact, DeliveryPhone: d.DeliveryPhone,
		BahtText: func() string {
			if d.Type == "DELIVERY_ORDER" {
				return formatThaiBahtText(d.TotalAmount)
			}
			return ""
		}(),

		ShowDiscount: p.ShowDiscount, ShowDeliveryBox: p.ShowDeliveryBox,
		ShowPayBox: p.ShowPayBox, PayCash: p.PayCash,

		Summary: buildSummary(d, p), Notes: derefStr(d.Notes),
		Banks: banks, QRURL: d.QRPaymentURL, BillRows: billRows,
		SigLeftTH: p.SigLeftTH, SigLeftEN: p.SigLeftEN,
		SigRightTH: p.SigRightTH, SigRightEN: p.SigRightEN,

		Pages: pages, IsSinglePage: totalPages == 1,
	}
}

// =============================================================================
//  5) TEMPLATE (HTML + CSS print-ready) — flexbox footer-pin, ไม่มี JS
// =============================================================================

var unifiedTmpl = template.Must(template.New("doc").Parse(unifiedDocHTML))

// RenderUnifiedDocumentHTML — entry point เดียวสำหรับทุกชนิดเอกสาร
// แทนที่ switch ใน RenderDocumentHTML เดิม ให้ทุก type เรียกฟังก์ชันนี้
func RenderUnifiedDocumentHTML(d DocData, store StoreInfo) (string, error) {
	vm := BuildDocumentView(d, store)
	var buf bytes.Buffer
	if err := unifiedTmpl.Execute(&buf, vm); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderOneCopy renders the document with a specific copy variant applied
// (Original/Copy badge + purpose tag + conditional signature block).
func renderOneCopy(d DocData, store StoreInfo, v doccopy.CopyVariant) (string, error) {
	vm := BuildDocumentView(d, store)
	vm.Badge = v.BadgeLabel()
	vm.Purpose = v.Purpose
	vm.ShowSignature = v.ShowSignature
	var buf bytes.Buffer
	if err := unifiedTmpl.Execute(&buf, vm); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var (
	bodyRe = regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	headRe = regexp.MustCompile(`(?is)<head[^>]*>(.*?)</head>`)
)

// RenderUnifiedDocumentCopies renders the full print set for a document type
// (per doccopy.SpecFor) as ONE HTML document: each copy is a page-broken sheet,
// every sheet carrying its own "ต้นฉบับ (Original)" / "สำเนา (Copy)" badge.
// A single browser print job therefore yields the whole legal copy set.
func RenderUnifiedDocumentCopies(d DocData, store StoreInfo, copyIdx int) (string, error) {
	specs := doccopy.SpecFor(d.Type)
	// copyIdx >= 0 selects a single copy (0-based); -1 (or out of range) = whole set.
	if copyIdx >= 0 && copyIdx < len(specs) {
		return renderOneCopy(d, store, specs[copyIdx])
	}
	if len(specs) == 0 {
		return RenderUnifiedDocumentHTML(d, store)
	}

	head := ""
	var sheets []string
	for _, v := range specs {
		html, err := renderOneCopy(d, store, v)
		if err != nil {
			return "", err
		}
		if head == "" {
			if m := headRe.FindStringSubmatch(html); m != nil {
				head = m[1]
			}
		}
		body := html
		if m := bodyRe.FindStringSubmatch(html); m != nil {
			body = m[1]
		}
		// Each copy is its own sheet. .copy-break forces a print page break between
		// copies AND adds a clear gap + divider on screen, so the preview drawer
		// visually separates ต้นฉบับ / สำเนา instead of running them together.
		sheets = append(sheets, fmt.Sprintf(`<div class="copy-break">%s</div>`, body))
	}
	return fmt.Sprintf(
		`<!DOCTYPE html><html lang="th"><head>%s</head><body>%s</body></html>`,
		head, strings.Join(sheets, "\n"),
	), nil
}

const unifiedDocHTML = `{{$root := .}}<!DOCTYPE html>
<html lang="th"><head><meta charset="utf-8"><title>{{.TitleTH}} {{.DocNo}}</title>
<style>
:root{ --row-h:7.5mm; --ink:#1a1a1a; --line:#bdbdbd; --muted:#666; }
*{ margin:0; padding:0; box-sizing:border-box; }
@page{ size:A4; margin:0; }
body{ font-family:'Sarabun','Tahoma',sans-serif; color:var(--ink); font-size:11px; background:#eee; -webkit-print-color-adjust:exact; print-color-adjust:exact; }

.page{ position:relative; width:210mm; min-height:297mm; padding:12mm 12mm 16mm; margin:0 auto 6mm; background:#fff; display:flex; flex-direction:column; page-break-after:always; }
.page:last-child{ page-break-after:auto; margin-bottom:0; }
/* copy-set: แต่ละ copy ใช้ .page เป็นตัวควบคุม page break เพียงจุดเดียว */
.copy-break{ page-break-after:auto; }
.copy-break:last-child{ page-break-after:auto; }
@media screen{ .copy-break:not(:last-child){ margin-bottom:16mm; border-bottom:2px dashed #94a3b8; } }
@media print{ body{background:#fff;} .page{ margin:0; box-shadow:none; } .copy-break{ display:contents; page-break-after:auto !important; } }

/* ---- footer-pin: spacer พองเฉพาะ "หน้าเดียว" → ลายเซ็นติดล่าง A4 ---- */
.doc-body{ flex:0 0 auto; }
.doc-spacer{ flex:0 0 auto; }
/* หน้าเดียว: spacer พอง → footer/ลายเซ็นถูกดันชิดล่าง A4.
   หน้า footer-only: ไม่ดันลง — ให้ footer อยู่ "ด้านบน" ต่อจากหัวย่อเลย (ช่องว่างไปอยู่ล่าง) */
.page.single .doc-spacer{ flex:1 1 auto; }
.doc-footer{ flex:0 0 auto; }

/* ---- header เต็ม ---- */
.hdr{ display:flex; justify-content:space-between; align-items:flex-start; padding-bottom:4mm; border-bottom:2px solid var(--ink); }
.hdr-left{ display:flex; gap:3mm; align-items:flex-start; }
.logo{ width:18mm; height:18mm; object-fit:contain; }
.store-name{ font-size:16px; font-weight:700; }
.store-meta{ font-size:10px; color:#333; line-height:1.45; }
.hdr-right{ text-align:right; min-width:62mm; }
.doc-title{ font-size:22px; font-weight:700; letter-spacing:.5px; }
.doc-id{ display:inline-block; border:1.2px solid var(--ink); border-radius:1mm; padding:.5mm 2mm; font-size:9px; font-weight:700; letter-spacing:.25px; margin-top:1mm; font-family:var(--mono); background:#fff; }
.doc-badge{ display:inline-block; border:1.2px solid var(--ink); border-radius:1mm; padding:.5mm 2mm; font-size:9px; font-weight:bold; margin-bottom:1mm; }
.doc-info-row{ display:flex; align-items:stretch; margin:0 0 3mm; }
.quotation-intro{ margin:0 0 3mm; font-size:11px; line-height:1.5; }
.quotation-intro strong{ font-weight:700; }
.doc-info-row .parties{ flex:6 1 0; min-width:0; margin:0; }
.doc-meta-panel{ flex:4 1 0; min-width:0; display:flex; align-items:stretch; border:1px solid var(--line); padding:0 3mm 2mm; }
.doc-meta{ width:100%; min-height:100%; padding:0; font-size:10px; line-height:1.5; }
.doc-meta .section-body{ gap:1mm; }
.doc-meta-line{ display:flex; align-items:flex-start; gap:2mm; min-width:0; line-height:1.5; }
.doc-meta-label{ flex:0 0 30mm; color:var(--muted); }
.doc-meta-value{ flex:1 1 auto; min-width:0; overflow-wrap:anywhere; font-weight:700; }
.doc-meta .b{ font-weight:700; margin-left:0; }

/* ---- header ย่อ (หน้าต่อ) ---- */
.hdr-mini{ display:flex; justify-content:space-between; align-items:baseline; padding-bottom:2mm; border-bottom:1px solid var(--ink); }
.m-store{ font-weight:700; }
.m-title{ font-size:10px; color:var(--muted); }

/* ---- กล่องลูกค้า/จัดส่ง ---- */
.parties{ display:flex; flex-direction:column; gap:0; margin:0; border:1px solid var(--line); border-right:0; min-width:0; }
.party{ flex:1 1 auto; min-width:0; border:0; padding:0 3mm 2mm; line-height:1.5; }
.combined-parties .section-body{ gap:.75mm; }
.combined-parties .info-row{ line-height:1.35; }
.header-section-title{ font-size:10px; font-weight:700; color:var(--ink); background:#f6f6f6; margin:0 -3mm 2mm; padding:1mm 3mm; line-height:1.5; border-bottom:1px solid var(--line); }
.section-body{ display:flex; flex-direction:column; gap:1mm; }
.info-row{ display:flex; align-items:flex-start; gap:2mm; min-width:0; line-height:1.5; }
.info-label{ flex:0 0 30mm; color:var(--muted); }
.info-value{ flex:1 1 auto; min-width:0; overflow-wrap:anywhere; }
.party-name{ font-weight:700; }
.party strong{ font-weight:700; color:var(--ink); }
.delivery-section{ font-size:9px; line-height:1.2; }
.delivery-section .section-body{ gap:.5mm; }
.delivery-section .info-row{ gap:1.5mm; line-height:1.2; }
.delivery-section .info-label{ flex-basis:25mm; }
.delivery-section .info-value{ line-height:1.2; }
.termsrow{ font-size:10px; margin:2mm 0 0; padding:0; border:0; background:transparent; line-height:1.5; }
.terms-title{ font-weight:700; margin-bottom:0.5mm; }
.baht-text-row{ display:flex; gap:2mm; margin:0; padding:1.5mm 3mm; border:1px solid var(--line); border-top:0; background:#f6f6f6; font-size:13px; line-height:1.5; break-before:avoid; page-break-before:avoid; }
.baht-text-row span:first-child{ color:var(--muted); }
.baht-text-row span:last-child{ margin-left:auto; text-align:right; font-size:13px; }

/* ---- ตารางสินค้า: full grid + คอลัมน์กึ่งกลาง (desc ชิดซ้าย) ---- */
.items{ width:100%; border-collapse:collapse; table-layout:fixed; border:1.2px solid var(--ink); }
.items th{ background:var(--ink); color:#fff; border:1px solid #444; padding:1.6mm 2mm; font-size:10px; text-align:center; -webkit-print-color-adjust:exact; print-color-adjust:exact; }
.items td{ height:var(--row-h); border:1px solid var(--line); padding:0.8mm 2mm; vertical-align:middle; text-align:center; overflow:hidden; }
.items .desc{ display:-webkit-box; -webkit-line-clamp:2; -webkit-box-orient:vertical; overflow:hidden; line-height:1.2; }
.items .desc .en{ color:var(--muted); font-size:10px; }
.items .left{ text-align:left; } .items .num{ text-align:right; }
.items tr.filler td{ height:var(--row-h); }
.col-no{ width:9mm; } .col-qty{ width:16mm; } .col-unit{ width:14mm; } .col-price,.col-disc,.col-amt{ width:22mm; }
.bill-items{ width:100%; border-collapse:collapse; table-layout:fixed; border:1.2px solid var(--ink); }
.bill-items th{ background:var(--ink); color:#fff; border:1px solid #444; padding:2.2mm 2mm; font-size:10px; text-align:center; }
.bill-items td{ height:var(--row-h); border:1px solid var(--line); padding:0.8mm 2mm; vertical-align:middle; text-align:center; }
.bill-items .left{ text-align:left; } .bill-items .num{ text-align:right; }
.bill-col-no{ width:14mm; } .bill-col-date{ width:38mm; } .bill-col-due{ width:38mm; } .bill-col-amount{ width:42mm; }
tr{ break-inside:avoid; } thead{ display:table-header-group; }

/* ---- footer: summary + paybox + remarks + signatures ---- */
.doc-footer{ margin-top:3mm; }
.foot-grid{ display:flex; gap:4mm; align-items:flex-start; }
.foot-grid.remarks-grid{ align-items:stretch; }
.paybox{ flex:1; border:1px solid var(--line); padding:2mm 3mm; font-size:10px; }
.pay-row{ display:flex; align-items:baseline; gap:0; margin-bottom:1.5mm; }
.pay-chk{ width:3.5mm; flex-shrink:0; }
.pay-lbl{ white-space:nowrap; flex-shrink:0; margin-right:1mm; }
.pay-dot{ flex:1; border-bottom:1px dashed var(--line); margin:0 1.5mm; position:relative; top:-1.5px; }
.pay-baht{ white-space:nowrap; flex-shrink:0; font-size:9px; }
.bank-sub{ display:flex; align-items:baseline; gap:1.5mm; margin:-0.5mm 0 1.5mm 3.5mm; font-size:10px; line-height:1.2; font-weight:700; color:var(--ink); }
.bank-sub-lbl{ white-space:nowrap; flex-shrink:0; }
.bank-sub-ln{ flex:1; border-bottom:1px solid var(--line); position:relative; top:-2px; }
.pay-cheque-area{ display:flex; gap:2mm; align-items:flex-start; margin-top:0.5mm; }
.pay-cheque-left{ flex:1; min-width:0; }
.pay-qr-col{ display:flex; flex-direction:column; align-items:center; flex-shrink:0; padding:0 2mm; }
.qr{ width:16mm; height:16mm; object-fit:contain; }
.qr-placeholder{ width:16mm; height:16mm; border:1px dashed var(--line); }
.qr-lbl{ font-size:8px; color:var(--muted); text-align:center; margin-top:0.5mm; }
.pay-cheque-right{ flex:1; min-width:0; }
.c-row{ display:flex; align-items:baseline; gap:1mm; margin-bottom:1.5mm; font-size:9px; }
.c-row2{ display:flex; gap:2mm; align-items:baseline; margin-bottom:1.5mm; font-size:9px; }
.c-lbl{ white-space:nowrap; flex-shrink:0; color:var(--muted); }
.c-line{ flex:1; border-bottom:1px solid var(--line); position:relative; top:-2px; }
.summary{ width:78mm; margin-left:auto; }
.sum-row{ display:flex; justify-content:space-between; padding:1mm 0; border-bottom:1px dashed var(--line); }
.sum-total{ border-top:2px solid var(--ink); border-bottom:none; font-size:16px; font-weight:700; margin-top:1mm; padding-top:2mm; }
.remarks{ margin-top:2mm; border:1px solid var(--line); padding:1.5mm 3mm; font-size:10px; min-height:9mm; }
.remarks-inline{ flex:1; min-width:0; height:100%; align-self:stretch; margin-top:0; }
.remarks .rh{ color:var(--muted); }
/* center the whole signature GROUP, and center the content INSIDE each column
   (otherwise the fixed-width underlines left-pack inside 56mm boxes and the
   visible ink drifts left of the page centerline despite justify-content:center) */
.signatures{ display:flex; justify-content:center; gap:14mm; margin-top:4mm; break-inside:avoid; }
.sig{ width:52mm; flex-shrink:0; }
.sig-t{ font-size:9px; font-weight:700; margin-bottom:2mm; text-align:center; border-bottom:1px solid var(--line); padding-bottom:.5mm; }
.sig-write{ display:flex; justify-content:center; align-items:baseline; gap:1mm; margin-top:4mm; margin-bottom:2mm; }
.sig-write-lbl{ font-size:9px; white-space:nowrap; flex-shrink:0; }
.sig-ln{ width:36mm; border-bottom:1px solid var(--ink); }
.sig-date-row{ display:flex; justify-content:center; align-items:baseline; gap:.8mm; font-size:8px; color:var(--muted); }
.sig-date-seg{ width:8mm; border-bottom:1px solid var(--ink); position:relative; top:-1px; }
.summary,.signatures,.remarks,.paybox{ break-inside:avoid; }

/* ---- page number: absolute ไม่กินพื้นที่ layout และไม่ล้นไปหน้าใหม่ ---- */
.page-no{ position:absolute; top:6mm; right:12mm; bottom:auto; font-size:9px; color:var(--muted); white-space:nowrap; }
</style></head>
<body>
{{range $pg := .Pages}}
<div class="page{{if $root.IsSinglePage}} single{{end}}{{if $pg.FooterOnly}} footer-only{{end}}">

  {{if $pg.IsFirst}}
  <header class="hdr">
    <div class="hdr-left">
      {{if $root.LogoURL}}<img class="logo" src="{{$root.LogoURL}}" alt="logo">{{end}}
      <div>
        <div class="store-name">{{$root.StoreName}}</div>
        {{if $root.StoreAddr}}<div class="store-meta">{{$root.StoreAddr}}</div>{{end}}
        {{if $root.StoreTaxID}}<div class="store-meta">เลขประจำตัวผู้เสียภาษี {{$root.StoreTaxID}}</div>{{end}}
        {{if $root.StorePhone}}<div class="store-meta">โทร. {{$root.StorePhone}}</div>{{end}}
        {{if $root.StoreBranch}}<div class="store-meta">สาขา {{$root.StoreBranch}}</div>{{end}}
      </div>
    </div>
    <div class="hdr-right">
      <div class="doc-title">{{$root.TitleTH}}</div>
      {{if $root.Badge}}<div class="doc-badge">{{$root.Badge}}</div>{{end}}
      <div class="doc-id">{{$root.DocNo}}</div>
    </div>
  </header>

  <section class="doc-info-row">
    <section class="parties combined-parties">
      <div class="party customer-section">
        <div class="header-section-title">{{if $root.ShowDeliveryBox}}ข้อมูลลูกค้าและจัดส่ง · CUSTOMER / DELIVERY{{else}}ข้อมูลลูกค้า / หน่วยงาน · CUSTOMER{{end}}</div>
        <div class="section-body">
          <div class="info-row"><span class="info-label">ชื่อหน่วยงาน</span><span class="info-value">{{$root.CustomerName}}</span></div>
          <div class="info-row"><span class="info-label">ที่อยู่</span><span class="info-value">{{if $root.CustomerAddr}}{{$root.CustomerAddr}}{{else}}-{{end}}</span></div>
          {{if and $root.DeliveryAddr (ne $root.DeliveryAddr $root.CustomerAddr)}}<div class="info-row"><span class="info-label">ที่อยู่จัดส่ง</span><span class="info-value">{{$root.DeliveryAddr}}</span></div>{{end}}
          {{if $root.DeliveryContact}}<div class="info-row"><span class="info-label">ผู้ติดต่อ / เซ็น</span><span class="info-value">{{$root.DeliveryContact}}</span></div>{{end}}
          {{if $root.CustomerPhone}}<div class="info-row"><span class="info-label">โทรศัพท์</span><span class="info-value">{{$root.CustomerPhone}}</span></div>{{else if $root.DeliveryPhone}}<div class="info-row"><span class="info-label">โทรศัพท์</span><span class="info-value">{{$root.DeliveryPhone}}</span></div>{{end}}
          <div class="info-row"><span class="info-label">เลขผู้เสียภาษี</span><span class="info-value">{{if $root.CustomerTaxID}}{{$root.CustomerTaxID}}{{else}}-{{end}}</span></div>
          {{if $root.DeliveryDate}}<div class="info-row"><span class="info-label">วันที่จัดส่ง</span><span class="info-value">{{$root.DeliveryDate}}</span></div>{{end}}
        </div>
      </div>
    </section>
    <div class="doc-meta-panel">
      <div class="doc-meta">
        <div class="header-section-title">รายละเอียดเอกสาร · DOCUMENT DETAILS</div>
        <div class="section-body">
          <div class="doc-meta-line"><span class="doc-meta-label">เลขที่เอกสาร</span><span class="doc-meta-value">{{$root.DocNo}}</span></div>
          <div class="doc-meta-line"><span class="doc-meta-label">วันที่ออกเอกสาร</span><span class="doc-meta-value">{{$root.DocDate}}</span></div>
          {{if $root.PriceValidity}}<div class="doc-meta-line"><span class="doc-meta-label">ระยะเวลาใบเสนอราคา</span><span class="doc-meta-value">{{$root.PriceValidity}}</span></div>{{end}}
          {{if $root.PaymentTerms}}<div class="doc-meta-line"><span class="doc-meta-label">เงื่อนไขการชำระเงิน</span><span class="doc-meta-value">ภายใน {{$root.PaymentTerms}}</span></div>{{end}}
          {{if $root.DeliveryLeadTime}}<div class="doc-meta-line"><span class="doc-meta-label">กำหนดส่งสินค้า</span><span class="doc-meta-value">ภายใน {{$root.DeliveryLeadTime}} หลังได้รับใบสั่งซื้อ</span></div>{{end}}
          {{if $root.ShowDeliveryBox}}{{if $root.PORefNo}}<div class="doc-meta-line"><span class="doc-meta-label">อ้างอิงใบสั่งซื้อ</span><span class="doc-meta-value">{{$root.PORefNo}}</span></div>{{end}}{{if $root.RefNo}}<div class="doc-meta-line"><span class="doc-meta-label">{{$root.RefLabel}}</span><span class="doc-meta-value">{{$root.RefNo}}</span></div>{{end}}{{end}}
          {{if $root.QuotationRefNo}}<div class="doc-meta-line"><span class="doc-meta-label">อ้างอิงใบเสนอราคา</span><span class="doc-meta-value">{{$root.QuotationRefNo}}</span></div>{{end}}
          {{if $root.SpecialLabel}}<div class="doc-meta-line"><span class="doc-meta-label">{{$root.SpecialLabel}}</span><span class="doc-meta-value">{{$root.SpecialValue}}</span></div>{{end}}
          {{if $root.StaffName}}<div class="doc-meta-line"><span class="doc-meta-label">ผู้ออกเอกสาร</span><span class="doc-meta-value">{{$root.StaffName}}</span></div>{{end}}
        </div>
      </div>
    </div>
  </section>
  {{if and (eq $root.TitleTH "ใบเสนอราคา") $root.QuotationSummary}}
  <div class="quotation-intro"><strong>{{$root.StoreName}}</strong> ขอเสนอราคา <strong>{{$root.QuotationSummary}}</strong> เพื่อโปรดพิจารณา โดยมีรายละเอียดดังต่อไปนี้</div>
  {{end}}

  {{else}}
  <header class="hdr-mini">
    <span class="m-store">{{$root.StoreName}}</span>
    <span class="m-title">{{$root.TitleTH}} · {{$root.DocNo}} (ต่อ / cont.)</span>
  </header>
  {{end}}

  {{if not $pg.FooterOnly}}
  <div class="doc-body">
    {{if eq $root.TitleTH "ใบวางบิล"}}
    <table class="bill-items">
      <thead><tr>
        <th class="bill-col-no">ลำดับ</th>
        <th class="left">ใบส่งสินค้า (DO)</th>
        <th class="bill-col-date">วันที่ออกเอกสาร</th>
        <th class="bill-col-due">วันครบกำหนด</th>
        <th class="bill-col-amount">จำนวนเงิน</th>
      </tr></thead>
      <tbody>
      {{range $row := $root.BillRows}}
        <tr><td>{{$row.No}}</td><td class="left">ใบส่งสินค้าเลขที่ {{$row.DocumentNo}}</td><td>{{$row.IssueDate}}</td><td>{{$row.DueDate}}</td><td class="num">{{$row.Amount}}</td></tr>
      {{end}}
      {{range $pg.FillerRows}}
        <tr class="filler"><td>&nbsp;</td><td class="left"></td><td></td><td></td><td class="num"></td></tr>
      {{end}}
      </tbody>
    </table>
    {{else}}
    <table class="items">
      <thead><tr>
        <th class="col-no">ลำดับ</th>
        <th class="left">รายการสินค้า (Description)</th>
        <th class="col-qty">จำนวน</th>
        <th class="col-unit">หน่วย</th>
        <th class="col-price">ราคา/หน่วย</th>
        {{if $root.ShowDiscount}}<th class="col-disc">ส่วนลด</th>{{end}}
        <th class="col-amt">จำนวนเงิน</th>
      </tr></thead>
      <tbody>
      {{range $it := $pg.Items}}
        <tr>
          <td>{{$it.No}}</td>
          <td class="left"><div class="desc">{{$it.Desc}}{{if $it.DescEn}}<br><span class="en">({{$it.DescEn}})</span>{{end}}</div></td>
          <td>{{$it.Qty}}</td>
          <td>{{$it.Unit}}</td>
          <td class="num">{{$it.UnitPrice}}</td>
          {{if $root.ShowDiscount}}<td class="num">{{if $it.HasDiscount}}{{$it.Discount}}{{else}}-{{end}}</td>{{end}}
          <td class="num">{{$it.Amount}}</td>
        </tr>
      {{end}}
      {{range $pg.FillerRows}}
        <tr class="filler">
          <td>&nbsp;</td><td class="left"></td><td></td><td></td><td class="num"></td>{{if $root.ShowDiscount}}<td class="num"></td>{{end}}<td class="num"></td>
        </tr>
      {{end}}
      </tbody>
    </table>
    {{end}}
  </div>
  {{end}}

  {{if and $pg.IsLast $root.BahtText}}<div class="baht-text-row"><span>จำนวนเงินตัวอักษร / Text:</span><span>({{$root.BahtText}})</span></div>{{end}}

  <div class="doc-spacer"></div>

  {{if $pg.IsLast}}
  <footer class="doc-footer">
    <div class="foot-grid{{if not $root.ShowPayBox}} remarks-grid{{end}}">
      {{if $root.ShowPayBox}}
      <div class="paybox">
        <div class="pay-row"><span class="pay-chk">{{if $root.PayCash}}&#9745;{{else}}&#9744;{{end}}</span><span class="pay-lbl">เงินสด (Cash)</span><span class="pay-dot"></span><span class="pay-baht">บาท</span></div>
        <div class="pay-row"><span class="pay-chk">&#9744;</span><span class="pay-lbl">โอนเงินเข้าบัญชี (Bank Transfer)</span><span class="pay-dot"></span><span class="pay-baht">บาท</span></div>
        {{if $root.Banks}}
          {{range $bank := $root.Banks}}
          <div class="bank-sub"><span class="bank-sub-lbl">{{$bank.Name}}</span><span class="bank-sub-lbl">เลขบัญชี {{$bank.No}}</span><span class="bank-sub-lbl">{{$bank.Holder}}</span></div>
          {{end}}
        {{else}}
          <div class="bank-sub"><span class="bank-sub-lbl">ธนาคาร</span><span class="bank-sub-ln"></span><span class="bank-sub-lbl">เลขบัญชี</span><span class="bank-sub-ln"></span></div>
        {{end}}
        <div class="pay-row"><span class="pay-chk">&#9744;</span><span class="pay-lbl">เครดิต (Credit)</span><span class="pay-dot"></span><span class="pay-baht">บาท</span></div>
        <div class="pay-cheque-area">
          <div class="pay-cheque-left"><div class="pay-row"><span class="pay-chk">&#9744;</span><span class="pay-lbl">เช็ค (Cheque)</span><span class="pay-dot"></span><span class="pay-baht">บาท</span></div></div>
          <div class="pay-qr-col">{{if $root.QRURL}}<img class="qr" src="{{$root.QRURL}}" alt="QR PromptPay">{{else}}<div class="qr-placeholder"></div>{{end}}<div class="qr-lbl">PromptPay</div></div>
          <div class="pay-cheque-right">
            <div class="c-row"><span class="c-lbl">ผู้รับเงิน / Payee</span><span class="c-line"></span></div>
            <div class="c-row2"><span class="c-lbl">วันที่</span><span class="c-line"></span><span class="c-lbl">ธนาคาร</span><span class="c-line"></span></div>
            <div class="c-row"><span class="c-lbl">เลขที่เช็ค / No.</span><span class="c-line"></span></div>
            <div class="c-row"><span class="c-lbl">ลงวันที่ / Date</span><span class="c-line"></span></div>
          </div>
        </div>
      </div>
      {{else}}
      <div class="remarks remarks-inline"><span class="rh">หมายเหตุ (Remarks):</span> {{$root.Notes}}</div>
      {{end}}
      <div class="summary">
        {{range $root.Summary}}
        <div class="sum-row{{if .Strong}} sum-total{{end}}"><span>{{.Label}}</span><span>{{.Value}}</span></div>
        {{end}}
      </div>
    </div>

    {{if $root.ShowPayBox}}<div class="remarks"><span class="rh">หมายเหตุ (Remarks):</span> {{$root.Notes}}</div>{{end}}
    {{if $root.PriceTerms}}<section class="termsrow"><div class="terms-title">เงื่อนไข (Terms)</div><div>1. {{$root.PriceTerms}}</div><div>2. {{$root.DeliveryTerms}}</div></section>{{end}}

    {{if $root.ShowSignature}}
    <div class="signatures">
      <div class="sig">
        <div class="sig-t">{{$root.SigLeftTH}} / {{$root.SigLeftEN}}</div>
        <div class="sig-write"><span class="sig-write-lbl">ลงชื่อ</span><span class="sig-ln"></span></div>
        <div class="sig-date-row"><span>วันที่</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span></div>
      </div>
      <div class="sig">
        <div class="sig-t">{{$root.SigRightTH}} / {{$root.SigRightEN}}</div>
        <div class="sig-write"><span class="sig-write-lbl">ลงชื่อ</span><span class="sig-ln"></span></div>
        <div class="sig-date-row"><span>วันที่</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span></div>
      </div>
    </div>
    {{end}}
  </footer>
  {{end}}

  <div class="page-no">หน้า {{$pg.PageNo}} / {{$pg.TotalPages}} · Page {{$pg.PageNo}} of {{$pg.TotalPages}}</div>
</div>
{{end}}
</body></html>`
