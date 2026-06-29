package docpdf

import (
	"bytes"
	"fmt"
	"math"
	"time"

	"github.com/jung-kurt/gofpdf"

	"pos-backend/internal/platform/doccopy"
)

const (
	pageW  = 210.0
	pageH  = 297.0
	margin = 20.0
	body   = pageW - margin*2 // 170 mm usable width
)

// RenderInvoicePDF generates an A4 Invoice PDF and returns the raw bytes.
//
// All financial figures are taken VERBATIM from the stored document totals
// (Subtotal / TotalDiscount / VATAmount / TotalAmount) — the builder never
// recomputes subtotal, discount or VAT. This guarantees the Invoice PDF shows
// the same numbers as the receipt, the HTML document and the stored sale.
func RenderInvoicePDF(in InvoicePDFInput) ([]byte, error) {
	return RenderInvoicePDFCopies(in, nil)
}

// RenderInvoicePDFCopies renders one A4 page per copy variant into a SINGLE PDF
// (Original + Copy set), each page stamped with its "ต้นฉบับ/สำเนา" badge per the
// Thai Revenue copy rules. nil/empty variants → a single page (no badge), keeping
// the original single-document behaviour.
func RenderInvoicePDFCopies(in InvoicePDFInput, variants []doccopy.CopyVariant) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)
	font := registerFont(pdf)

	if len(variants) == 0 {
		pdf.AddPage()
		drawInvoicePage(pdf, font, in)
	} else {
		for _, v := range variants {
			pi := in // copy: per-page badge/purpose/signature
			pi.BadgeText = v.BadgeLabel()
			pi.PurposeText = v.Purpose
			pi.ShowSignature = v.ShowSignature
			pdf.AddPage()
			drawInvoicePage(pdf, font, pi)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf render: %w", err)
	}
	return buf.Bytes(), nil
}

// drawInvoicePage draws one complete document page onto the current pdf page.
func drawInvoicePage(pdf *gofpdf.Fpdf, font string, in InvoicePDFInput) {
	// ── Stored totals (no recompute) ──────────────────────────────────────────
	subtotal := in.Subtotal
	discountAmt := in.TotalDiscount
	vatAmt := in.VATAmount
	totalDue := in.TotalAmount

	// ── 1. Header band ────────────────────────────────────────────────────────
	titleBand := "ใบแจ้งหนี้  /  Invoice"
	if in.DocTitleTH != "" {
		titleBand = in.DocTitleTH + "  /  " + in.DocTitleEN
	}
	pdf.SetFillColor(109, 40, 217)
	pdf.Rect(0, 0, pageW, 12, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(font, "", 8)
	pdf.SetXY(margin, 3.5)
	pdf.CellFormat(body, 5, titleBand, "", 1, "R", false, 0, "")

	// ── 1b. Logo + seller name row ────────────────────────────────────────────
	const logoW, logoH = 50.0, 12.5 // 480px×120px ratio at PDF scale
	nameX := margin
	if len(in.SellerLogoBytes) > 0 {
		ext := in.SellerLogoExt
		if ext == "" {
			ext = "png"
		}
		r := bytes.NewReader(in.SellerLogoBytes)
		pdf.RegisterImageOptionsReader("seller_logo", gofpdf.ImageOptions{ImageType: ext}, r)
		pdf.ImageOptions("seller_logo", margin, 14, logoW, logoH, false, gofpdf.ImageOptions{}, 0, "")
		nameX = margin + logoW + 3
	}

	// ── 2. Seller name (to the right of logo) ─────────────────────────────────
	pdf.SetTextColor(30, 27, 75)
	pdf.SetFont(font, "", 11)
	pdf.SetXY(nameX, 14)
	nameW := body/2 - (nameX - margin)
	pdf.CellFormat(nameW, 6, in.SellerName, "", 1, "L", false, 0, "")
	pdf.SetFont(font, "", 8)
	pdf.SetTextColor(100, 116, 139)
	if in.SellerAddress != "" {
		pdf.SetX(nameX)
		pdf.CellFormat(nameW, 4.5, in.SellerAddress, "", 1, "L", false, 0, "")
	}
	if in.SellerPhone != "" {
		pdf.SetX(nameX)
		pdf.CellFormat(nameW, 4.5, "โทร: "+in.SellerPhone, "", 1, "L", false, 0, "")
	}
	if in.SellerTaxID != "" {
		pdf.SetX(nameX)
		pdf.CellFormat(nameW, 4.5, "TIN: "+in.SellerTaxID, "", 1, "L", false, 0, "")
	}

	// ── 2b. Document title ────────────────────────────────────────────────────
	bigTitle := "INVOICE"
	if in.DocTitleEN != "" {
		bigTitle = in.DocTitleEN
	}
	pdf.SetTextColor(30, 27, 75)
	pdf.SetXY(margin, 30)
	pdf.SetFont(font, "", 20)
	pdf.CellFormat(body/2, 12, bigTitle, "", 0, "L", false, 0, "")

	// ── 2c. Copy badge (ต้นฉบับ/สำเนา) — top-right corner per Revenue rules ─────
	metaY := 17.0
	if in.BadgeText != "" {
		bw := 42.0
		bx := pageW - margin - bw
		pdf.SetDrawColor(30, 27, 75)
		pdf.SetLineWidth(0.4)
		pdf.SetFont(font, "", 10)
		pdf.SetTextColor(30, 27, 75)
		pdf.SetXY(bx, 14.5)
		pdf.CellFormat(bw, 6, in.BadgeText, "1", 1, "C", false, 0, "")
		if in.PurposeText != "" {
			pdf.SetFont(font, "", 7)
			pdf.SetTextColor(110, 116, 139)
			pdf.SetXY(bx, 21)
			pdf.CellFormat(bw, 4, in.PurposeText, "", 1, "C", false, 0, "")
		}
		pdf.SetTextColor(30, 27, 75)
		pdf.SetLineWidth(0.2)
		metaY = 26.0
	}

	// Right: doc number, dates (pushed below the badge when present)
	pdf.SetFont(font, "", 9)
	pdf.SetXY(margin+body/2, metaY)
	pdf.CellFormat(body/4, 6, "เลขที่เอกสาร / Doc No.", "", 0, "R", false, 0, "")
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

	// ── 3. Customer info ─────────────────────────────────────────────────────
	yBefore := pdf.GetY()

	sectionHeader(pdf, font, "ผู้รับเอกสาร / To", margin, yBefore, body)
	y := pdf.GetY() + 1
	pdf.SetFont(font, "", 9)
	infoRow(pdf, font, margin, y, body/2, in.CustomerName)
	y = pdf.GetY()
	if in.CustomerAddress != "" {
		infoRow(pdf, font, margin, y, body/2, in.CustomerAddress)
		y = pdf.GetY()
	}
	if in.CustomerTaxID != "" {
		infoRow(pdf, font, margin, y, body/2, "เลขผู้เสียภาษี: "+in.CustomerTaxID)
		y = pdf.GetY()
	}
	if in.CreditTerm > 0 {
		infoRow(pdf, font, margin, y, body/2, fmt.Sprintf("เครดิต: %d วัน", in.CreditTerm))
	}

	if pdf.GetY() > yBefore {
		pdf.SetY(pdf.GetY())
	}
	pdf.Ln(5)
	hRule(pdf)
	pdf.Ln(3)

	// ── 4. Line items table ───────────────────────────────────────────────────
	cNo := 8.0
	cDesc := 60.0
	cQty := 16.0
	cUnit := 16.0
	cUnitPrice := 24.0
	cDisc := 22.0
	cTotal := 24.0
	// cNo+cDesc+cQty+cUnit+cUnitPrice+cDisc+cTotal = 170 = body ✓

	// Header row
	pdf.SetFillColor(237, 233, 254) // violet-100
	pdf.SetTextColor(88, 28, 135)   // violet-900
	pdf.SetFont(font, "", 8)
	rowH := 7.0
	pdf.CellFormat(cNo, rowH, "#", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cDesc, rowH, "รายการ / Description", "TB", 0, "L", true, 0, "")
	pdf.CellFormat(cQty, rowH, "จำนวน", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cUnit, rowH, "หน่วย", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cUnitPrice, rowH, "ราคา/หน่วย", "TB", 0, "R", true, 0, "")
	pdf.CellFormat(cDisc, rowH, "ส่วนลด", "TB", 0, "R", true, 0, "")
	pdf.CellFormat(cTotal, rowH, "รวม", "TB", 1, "R", true, 0, "")

	pdf.SetTextColor(51, 65, 85) // slate-700
	pdf.SetFont(font, "", 9)
	for i, it := range in.Items {
		// Stored post-discount line total; never recompute Quantity*UnitPrice.
		lineTotal := it.LineAmount
		discCell := "-"
		if it.LineDiscount > 0 {
			discCell = "-" + money(it.LineDiscount)
		}
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
		pdf.CellFormat(cDisc, rowH, discCell, "B", 0, "R", fill, 0, "")
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
			"ส่วนลด / Discount", "-"+money(discountAmt), false, false)
		summaryRow(pdf, font, sumX, sumLabelW, sumValueW,
			"ยอดก่อนภาษี / Pre-VAT", money(in.PreVatAmount), false, false)
	}
	if in.VATRate > 0 {
		summaryRow(pdf, font, sumX, sumLabelW, sumValueW,
			fmt.Sprintf("ภาษีมูลค่าเพิ่ม %g%% / VAT", in.VATRate), money(vatAmt), false, false)
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

	// ── 7. Goods-received signature block (company copy only) ────────────────
	if in.ShowSignature {
		drawInvoiceSignature(pdf, font)
	}
}

// drawInvoiceSignature draws the two-column delivered-by / received-by signature
// block at the bottom of the page.
func drawInvoiceSignature(pdf *gofpdf.Fpdf, font string) {
	pdf.Ln(10)
	y := pdf.GetY()
	if y > pageH-40 {
		y = pageH - 40
	}
	colW := body / 2
	for i, label := range []string{"ผู้ส่งสินค้า / Delivered By", "ผู้รับสินค้า / Received By"} {
		x := margin + float64(i)*colW
		lineY := y + 12
		pdf.SetDrawColor(120, 120, 120)
		pdf.SetLineWidth(0.2)
		pdf.Line(x+6, lineY, x+colW-12, lineY)
		pdf.SetXY(x, lineY+1)
		pdf.SetFont(font, "", 8)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(colW-6, 5, label, "", 0, "C", false, 0, "")
		pdf.SetXY(x, lineY+6)
		pdf.CellFormat(colW-6, 5, "วันที่ / Date ......./......./.......", "", 0, "C", false, 0, "")
	}
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
