package document

import (
	"bytes"
	"fmt"
	"math"
	"time"

	"github.com/jung-kurt/gofpdf"
)

const (
	pageW  = 210.0
	pageH  = 297.0
	margin = 20.0
	body   = pageW - margin*2 // 170 mm usable width
)

// RenderInvoicePDF generates an A4 Invoice PDF and returns the raw bytes.
func RenderInvoicePDF(in InvoicePDFInput) ([]byte, error) {
	// ── Calculations ──────────────────────────────────────────────────────────
	var subtotal float64
	for _, it := range in.Items {
		subtotal += it.Quantity * it.UnitPrice
	}
	discountAmt := math.Round(subtotal*in.DiscountPercent/100*100) / 100
	afterDiscount := subtotal - discountAmt
	var vatAmt float64
	if in.VATRegistered {
		vatAmt = math.Round(afterDiscount*0.07*100) / 100
	}
	totalDue := afterDiscount + vatAmt

	// ── PDF setup ─────────────────────────────────────────────────────────────
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)
	pdf.AddPage()
	font := registerFont(pdf)

	// ── 1. Header band ────────────────────────────────────────────────────────
	pdf.SetFillColor(109, 40, 217) // violet-700
	pdf.Rect(0, 0, pageW, 12, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(font, "", 8)
	pdf.SetXY(margin, 3.5)
	pdf.CellFormat(body/2, 5, in.SellerName, "", 0, "L", false, 0, "")
	pdf.CellFormat(body/2, 5, "ใบแจ้งหนี้  /  Invoice", "", 1, "R", false, 0, "")

	// ── 2. Title block ────────────────────────────────────────────────────────
	pdf.SetTextColor(30, 27, 75) // indigo-950
	pdf.SetXY(margin, 16)
	pdf.SetFont(font, "", 22)
	pdf.CellFormat(body/2, 12, "INVOICE", "", 0, "L", false, 0, "")

	// Right: doc number, dates
	pdf.SetFont(font, "", 9)
	pdf.SetXY(margin+body/2, 17)
	pdf.CellFormat(body/4, 6, "เลขที่เอกสาร / Invoice No.", "", 0, "R", false, 0, "")
	pdf.SetFont(font, "", 9)
	pdf.CellFormat(body/4, 6, in.InvoiceNo, "", 1, "R", false, 0, "")

	pdf.SetX(margin + body/2)
	pdf.SetFont(font, "", 8)
	pdf.CellFormat(body/4, 5, "วันที่ออก / Issue Date", "", 0, "R", false, 0, "")
	pdf.CellFormat(body/4, 5, thaiDate(in.IssueDate), "", 1, "R", false, 0, "")

	pdf.SetX(margin + body/2)
	pdf.CellFormat(body/4, 5, "วันครบกำหนด / Due Date", "", 0, "R", false, 0, "")
	pdf.CellFormat(body/4, 5, thaiDate(in.DueDate), "", 1, "R", false, 0, "")

	if in.ReferenceDO != "" {
		pdf.SetX(margin + body/2)
		pdf.CellFormat(body/4, 5, "อ้างอิง DO / Ref. DO", "", 0, "R", false, 0, "")
		pdf.CellFormat(body/4, 5, in.ReferenceDO, "", 1, "R", false, 0, "")
	}

	pdf.Ln(4)
	hRule(pdf)
	pdf.Ln(3)

	// ── 3. Seller / Customer columns ─────────────────────────────────────────
	colW := body / 2
	yBefore := pdf.GetY()

	// Seller (left)
	sectionHeader(pdf, font, "ผู้ออกเอกสาร / From", margin, yBefore, colW-5)
	y := pdf.GetY() + 1
	pdf.SetFont(font, "", 9)
	infoRow(pdf, font, margin, y, colW-5, in.SellerName)
	y = pdf.GetY()
	if in.SellerAddress != "" {
		infoRow(pdf, font, margin, y, colW-5, in.SellerAddress)
		y = pdf.GetY()
	}
	if in.SellerPhone != "" {
		infoRow(pdf, font, margin, y, colW-5, "โทร: "+in.SellerPhone)
		y = pdf.GetY()
	}
	if in.SellerTaxID != "" {
		infoRow(pdf, font, margin, y, colW-5, "เลขผู้เสียภาษี: "+in.SellerTaxID)
	}

	yAfterSeller := pdf.GetY()

	// Customer (right)
	xRight := margin + colW
	sectionHeader(pdf, font, "ผู้รับเอกสาร / To", xRight, yBefore, colW)
	y = pdf.GetY() + 1
	pdf.SetFont(font, "", 9)
	infoRow(pdf, font, xRight, y, colW, in.CustomerName)
	y = pdf.GetY()
	if in.CustomerAddress != "" {
		infoRow(pdf, font, xRight, y, colW, in.CustomerAddress)
		y = pdf.GetY()
	}
	if in.CustomerTaxID != "" {
		infoRow(pdf, font, xRight, y, colW, "เลขผู้เสียภาษี: "+in.CustomerTaxID)
		y = pdf.GetY()
	}
	if in.CreditTerm > 0 {
		infoRow(pdf, font, xRight, y, colW, fmt.Sprintf("เครดิต: %d วัน", in.CreditTerm))
	}

	yAfterCustomer := pdf.GetY()
	if yAfterCustomer > yAfterSeller {
		pdf.SetY(yAfterCustomer)
	} else {
		pdf.SetY(yAfterSeller)
	}
	pdf.Ln(5)
	hRule(pdf)
	pdf.Ln(3)

	// ── 4. Line items table ───────────────────────────────────────────────────
	cNo := 8.0
	cDesc := 72.0
	cQty := 18.0
	cUnit := 20.0
	cUnitPrice := 26.0
	cTotal := 26.0
	// cNo+cDesc+cQty+cUnit+cUnitPrice+cTotal = 170 = body ✓

	// Header row
	pdf.SetFillColor(237, 233, 254) // violet-100
	pdf.SetTextColor(88, 28, 135)   // violet-900
	pdf.SetFont(font, "", 8)
	rowH := 7.0
	pdf.CellFormat(cNo, rowH, "#", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cDesc, rowH, "รายการ / Description", "TB", 0, "L", true, 0, "")
	pdf.CellFormat(cQty, rowH, "จำนวน", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cUnit, rowH, "หน่วย", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cUnitPrice, rowH, "ราคาต่อหน่วย", "TB", 0, "R", true, 0, "")
	pdf.CellFormat(cTotal, rowH, "รวม", "TB", 1, "R", true, 0, "")

	pdf.SetTextColor(51, 65, 85) // slate-700
	pdf.SetFont(font, "", 9)
	for i, it := range in.Items {
		lineTotal := it.Quantity * it.UnitPrice
		fill := i%2 == 1
		if fill {
			pdf.SetFillColor(250, 248, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(cNo, rowH, fmt.Sprintf("%d", i+1), "B", 0, "C", fill, 0, "")
		pdf.CellFormat(cDesc, rowH, it.Description, "B", 0, "L", fill, 0, "")
		pdf.CellFormat(cQty, rowH, fmtQtyPDF(it.Quantity), "B", 0, "C", fill, 0, "")
		pdf.CellFormat(cUnit, rowH, it.Unit, "B", 0, "C", fill, 0, "")
		pdf.CellFormat(cUnitPrice, rowH, money(it.UnitPrice), "B", 0, "R", fill, 0, "")
		pdf.CellFormat(cTotal, rowH, money(lineTotal), "B", 1, "R", fill, 0, "")
	}
	pdf.Ln(4)

	// ── 5. Summary block (right-aligned) ─────────────────────────────────────
	sumLabelW := 45.0
	sumValueW := 30.0
	sumX := pageW - margin - sumLabelW - sumValueW

	pdf.SetTextColor(100, 116, 139) // slate-500
	pdf.SetFont(font, "", 9)

	summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ยอดรวม / Subtotal", money(subtotal), false, false)

	if discountAmt > 0 {
		summaryRow(pdf, font, sumX, sumLabelW, sumValueW,
			fmt.Sprintf("ส่วนลด %.2f%% / Discount", in.DiscountPercent),
			"-"+money(discountAmt), false, false)
	}
	if in.VATRegistered {
		summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ภาษีมูลค่าเพิ่ม 7% / VAT", money(vatAmt), false, false)
	}

	// Total due — bold + accent
	pdf.Ln(1)
	pdf.SetDrawColor(109, 40, 217)
	pdf.SetLineWidth(0.5)
	startX := sumX
	pdf.Line(startX, pdf.GetY(), pageW-margin, pdf.GetY())
	pdf.Ln(1)
	summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ยอดที่ต้องชำระ / Total Due", money(totalDue), true, true)

	pdf.Ln(6)
	hRule(pdf)
	pdf.Ln(4)

	// ── 6. Footer: bank / PromptPay / note ───────────────────────────────────
	if in.BankName != "" || in.AccountNumber != "" || in.PromptPay != "" || in.Note != "" {
		pdf.SetTextColor(30, 27, 75)
		pdf.SetFont(font, "", 8)
		pdf.SetX(margin)
		pdf.CellFormat(body/2, 5, "ข้อมูลการชำระเงิน / Payment Details", "", 1, "L", false, 0, "")
		pdf.SetFont(font, "", 9)
		if in.BankName != "" {
			footerLine(pdf, font, "ธนาคาร / Bank:", in.BankName)
		}
		if in.AccountNumber != "" {
			footerLine(pdf, font, "เลขบัญชี / Account:", in.AccountNumber)
		}
		if in.PromptPay != "" {
			footerLine(pdf, font, "พร้อมเพย์ / PromptPay:", in.PromptPay)
		}
		if in.Note != "" {
			pdf.Ln(2)
			pdf.SetFont(font, "", 8)
			pdf.SetTextColor(100, 116, 139)
			pdf.SetX(margin)
			pdf.MultiCell(body, 5, "หมายเหตุ: "+in.Note, "", "L", false)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf render: %w", err)
	}
	return buf.Bytes(), nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func hRule(pdf *gofpdf.Fpdf) {
	pdf.SetDrawColor(196, 181, 253) // violet-300
	pdf.SetLineWidth(0.3)
	y := pdf.GetY()
	pdf.Line(margin, y, pageW-margin, y)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.2)
}

func sectionHeader(pdf *gofpdf.Fpdf, font, label string, x, y, w float64) {
	pdf.SetFont(font, "", 7.5)
	pdf.SetTextColor(109, 40, 217)
	pdf.SetXY(x, y)
	pdf.CellFormat(w, 5, label, "", 1, "L", false, 0, "")
	pdf.SetTextColor(51, 65, 85)
}

func infoRow(pdf *gofpdf.Fpdf, font string, x, y, w float64, text string) {
	pdf.SetXY(x, y)
	pdf.SetFont(font, "", 9)
	pdf.MultiCell(w, 5, text, "", "L", false)
}

func summaryRow(pdf *gofpdf.Fpdf, font string, x, labelW, valueW float64, label, value string, bold, accent bool) {
	pdf.SetX(x)
	if bold {
		pdf.SetFont(font, "", 10)
		if accent {
			pdf.SetTextColor(109, 40, 217)
		} else {
			pdf.SetTextColor(30, 27, 75)
		}
	} else {
		pdf.SetFont(font, "", 9)
		pdf.SetTextColor(100, 116, 139)
	}
	pdf.CellFormat(labelW, 6, label, "", 0, "L", false, 0, "")
	pdf.CellFormat(valueW, 6, value, "", 1, "R", false, 0, "")
	pdf.SetTextColor(51, 65, 85)
}

func footerLine(pdf *gofpdf.Fpdf, font, label, value string) {
	pdf.SetX(margin)
	pdf.SetFont(font, "", 8)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(35, 5, label, "", 0, "L", false, 0, "")
	pdf.SetTextColor(30, 27, 75)
	pdf.CellFormat(body-35, 5, value, "", 1, "L", false, 0, "")
}

func money(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	// integer and fractional parts
	intPart := int64(v)
	frac := int64(math.Round((v-float64(intPart))*100))
	if frac == 100 {
		intPart++
		frac = 0
	}
	// comma-group the integer part
	s := fmt.Sprintf("%d", intPart)
	out := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(ch)
	}
	result := fmt.Sprintf("%s.%02d", out, frac)
	if neg {
		return "-" + result
	}
	return result
}

func thaiDate(t time.Time) string {
	return fmt.Sprintf("%02d/%02d/%d", t.Day(), int(t.Month()), t.Year()+543)
}

func fmtQtyPDF(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%d", int64(q))
	}
	return fmt.Sprintf("%.2f", q)
}
