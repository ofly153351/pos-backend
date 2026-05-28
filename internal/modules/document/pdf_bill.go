package document

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// BillPDFInput holds the data for a Bill PDF (ใบวางบิล).
type BillPDFInput struct {
	SellerName    string
	SellerAddress string
	SellerTaxID   string
	SellerPhone   string

	CustomerName    string
	CustomerAddress string
	CustomerTaxID   string

	BillNo    string
	IssueDate time.Time
	DueDate   *time.Time // nil → "ไม่ระบุ"

	Items       []InvoicePDFItem
	Subtotal    float64
	VATRate     float64
	VATAmount   float64
	TotalAmount float64
	Note        string
}

// RenderBillPDF generates an A4 Bill (ใบวางบิล) PDF.
// Layout emphasises the due date and payment amount — distinct from RenderInvoicePDF.
func RenderBillPDF(in BillPDFInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)
	pdf.AddPage()
	font := registerFont(pdf)

	// ── 1. Header band ────────────────────────────────────────────────────────
	pdf.SetFillColor(109, 40, 217)
	pdf.Rect(0, 0, pageW, 12, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(font, "", 8)
	pdf.SetXY(margin, 3.5)
	pdf.CellFormat(body/2, 5, in.SellerName, "", 0, "L", false, 0, "")
	pdf.CellFormat(body/2, 5, "ใบวางบิล  /  Billing Notice", "", 1, "R", false, 0, "")

	// ── 2. Title block ────────────────────────────────────────────────────────
	pdf.SetTextColor(30, 27, 75)
	pdf.SetXY(margin, 16)
	pdf.SetFont(font, "", 22)
	pdf.CellFormat(body/2, 12, "BILL", "", 0, "L", false, 0, "")

	pdf.SetFont(font, "", 9)
	pdf.SetXY(margin+body/2, 17)
	pdf.CellFormat(body/4, 6, "เลขที่ / Bill No.", "", 0, "R", false, 0, "")
	pdf.CellFormat(body/4, 6, in.BillNo, "", 1, "R", false, 0, "")
	pdf.SetX(margin + body/2)
	pdf.SetFont(font, "", 8)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(body/4, 5, "ออกวันที่", "", 0, "R", false, 0, "")
	pdf.SetTextColor(30, 27, 75)
	pdf.CellFormat(body/4, 5, thaiDate(in.IssueDate), "", 1, "R", false, 0, "")

	pdf.Ln(4)
	hRule(pdf)
	pdf.Ln(2)

	// ── 3. Payment box (full-width prominent callout) ─────────────────────────
	pdf.SetFillColor(245, 243, 255) // violet-50
	pdf.SetDrawColor(196, 181, 253) // violet-300
	pdf.SetLineWidth(0.5)
	boxY := pdf.GetY()
	pdf.Rect(margin, boxY, body, 24, "FD")

	// Left half: due date
	pdf.SetXY(margin+4, boxY+2.5)
	pdf.SetFont(font, "", 7.5)
	pdf.SetTextColor(109, 40, 217)
	pdf.CellFormat(body/2-8, 5, "กรุณาชำระภายใน / Please Pay By", "", 1, "L", false, 0, "")
	pdf.SetXY(margin+4, boxY+8)
	pdf.SetFont(font, "", 13)
	pdf.SetTextColor(30, 27, 75)
	dueStr := "ไม่ระบุวันกำหนด"
	if in.DueDate != nil {
		dueStr = thaiDate(*in.DueDate)
	}
	pdf.CellFormat(body/2-8, 8, dueStr, "", 0, "L", false, 0, "")

	// Right half: amount
	pdf.SetXY(margin+body/2+2, boxY+2.5)
	pdf.SetFont(font, "", 7.5)
	pdf.SetTextColor(109, 40, 217)
	pdf.CellFormat(body/2-6, 5, "ยอดที่ต้องชำระ / Amount Due", "", 1, "R", false, 0, "")
	pdf.SetXY(margin+body/2+2, boxY+8)
	pdf.SetFont(font, "", 17)
	pdf.SetTextColor(76, 29, 149)
	pdf.CellFormat(body/2-6, 9, "฿"+money(in.TotalAmount), "", 0, "R", false, 0, "")

	pdf.SetY(boxY + 26)
	pdf.Ln(3)
	hRule(pdf)
	pdf.Ln(3)

	// ── 4. Customer / Seller columns ──────────────────────────────────────────
	colW := body / 2
	yBefore := pdf.GetY()

	sectionHeader(pdf, font, "เรียน / Billed To", margin, yBefore, colW-5)
	y := pdf.GetY() + 1
	pdf.SetFont(font, "", 9)
	infoRow(pdf, font, margin, y, colW-5, in.CustomerName)
	y = pdf.GetY()
	if in.CustomerAddress != "" {
		infoRow(pdf, font, margin, y, colW-5, in.CustomerAddress)
		y = pdf.GetY()
	}
	if in.CustomerTaxID != "" {
		infoRow(pdf, font, margin, y, colW-5, "เลขผู้เสียภาษี: "+in.CustomerTaxID)
	}
	yAfterCust := pdf.GetY()

	xRight := margin + colW
	sectionHeader(pdf, font, "ออกโดย / Issued By", xRight, yBefore, colW)
	y = pdf.GetY() + 1
	pdf.SetFont(font, "", 9)
	infoRow(pdf, font, xRight, y, colW, in.SellerName)
	y = pdf.GetY()
	if in.SellerAddress != "" {
		infoRow(pdf, font, xRight, y, colW, in.SellerAddress)
		y = pdf.GetY()
	}
	if in.SellerPhone != "" {
		infoRow(pdf, font, xRight, y, colW, "โทร: "+in.SellerPhone)
		y = pdf.GetY()
	}
	if in.SellerTaxID != "" {
		infoRow(pdf, font, xRight, y, colW, "เลขผู้เสียภาษี: "+in.SellerTaxID)
	}
	yAfterSeller := pdf.GetY()

	if yAfterSeller > yAfterCust {
		pdf.SetY(yAfterSeller)
	} else {
		pdf.SetY(yAfterCust)
	}
	pdf.Ln(4)
	hRule(pdf)
	pdf.Ln(3)

	// ── 5. Items table (no discount column — bills show net amounts) ──────────
	cNo, cDesc, cQty, cUnit, cPrice, cTotal := 8.0, 82.0, 18.0, 20.0, 21.0, 21.0

	pdf.SetFillColor(237, 233, 254)
	pdf.SetTextColor(88, 28, 135)
	pdf.SetFont(font, "", 8)
	rowH := 7.0
	pdf.CellFormat(cNo, rowH, "#", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cDesc, rowH, "รายการ / Description", "TB", 0, "L", true, 0, "")
	pdf.CellFormat(cQty, rowH, "จำนวน", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cUnit, rowH, "หน่วย", "TB", 0, "C", true, 0, "")
	pdf.CellFormat(cPrice, rowH, "ราคา/หน่วย", "TB", 0, "R", true, 0, "")
	pdf.CellFormat(cTotal, rowH, "รวม", "TB", 1, "R", true, 0, "")

	pdf.SetTextColor(51, 65, 85)
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
		pdf.CellFormat(cPrice, rowH, money(it.UnitPrice), "B", 0, "R", fill, 0, "")
		pdf.CellFormat(cTotal, rowH, money(lineTotal), "B", 1, "R", fill, 0, "")
	}
	pdf.Ln(3)

	// ── 6. Totals ─────────────────────────────────────────────────────────────
	sumLabelW := 40.0
	sumValueW := 28.0
	sumX := pageW - margin - sumLabelW - sumValueW

	pdf.SetTextColor(100, 116, 139)
	pdf.SetFont(font, "", 9)
	summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ยอดรวม / Subtotal", money(in.Subtotal), false, false)
	if in.VATRate > 0 {
		summaryRow(pdf, font, sumX, sumLabelW, sumValueW,
			fmt.Sprintf("VAT %.0f%%", in.VATRate), money(in.VATAmount), false, false)
	}
	pdf.Ln(1)
	pdf.SetDrawColor(109, 40, 217)
	pdf.SetLineWidth(0.5)
	pdf.Line(sumX, pdf.GetY(), pageW-margin, pdf.GetY())
	pdf.Ln(1)
	summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ยอดที่ต้องชำระ", money(in.TotalAmount), true, true)

	pdf.Ln(6)
	hRule(pdf)

	if in.Note != "" {
		pdf.Ln(3)
		pdf.SetX(margin)
		pdf.SetFont(font, "", 8)
		pdf.SetTextColor(100, 116, 139)
		pdf.MultiCell(body, 5, "หมายเหตุ: "+in.Note, "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("bill pdf render: %w", err)
	}
	return buf.Bytes(), nil
}
