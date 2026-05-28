package document

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// RenderStatementPDF generates an A4 Statement PDF and returns the raw bytes.
func RenderStatementPDF(in StatementPDFInput) ([]byte, error) {
	// ── Totals ────────────────────────────────────────────────────────────────
	var totalAmount, totalPaid, netBalance float64
	for _, row := range in.Invoices {
		totalAmount += row.Amount
		totalPaid += row.Paid
		netBalance += row.Balance
	}

	// ── PDF setup ─────────────────────────────────────────────────────────────
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
	pdf.CellFormat(body/2, 5, "ใบแจ้งยอด  /  Statement", "", 1, "R", false, 0, "")

	// ── 2. Title block ────────────────────────────────────────────────────────
	pdf.SetTextColor(30, 27, 75)
	pdf.SetXY(margin, 16)
	pdf.SetFont(font, "", 22)
	pdf.CellFormat(body/2, 12, "STATEMENT", "", 0, "L", false, 0, "")

	pdf.SetFont(font, "", 9)
	pdf.SetXY(margin+body/2, 17)
	pdf.CellFormat(body/4, 6, "เลขที่ / Statement No.", "", 0, "R", false, 0, "")
	pdf.CellFormat(body/4, 6, in.StatementNo, "", 1, "R", false, 0, "")
	pdf.SetX(margin + body/2)
	pdf.SetFont(font, "", 8)
	pdf.CellFormat(body/4, 5, "วันที่ออก / Issue Date", "", 0, "R", false, 0, "")
	pdf.CellFormat(body/4, 5, thaiDate(in.IssueDate), "", 1, "R", false, 0, "")
	pdf.SetX(margin + body/2)
	pdf.CellFormat(body/4, 5, "ช่วงเวลา / Period", "", 0, "R", false, 0, "")
	period := fmt.Sprintf("%s – %s", thaiDate(in.PeriodStart), thaiDate(in.PeriodEnd))
	pdf.CellFormat(body/4, 5, period, "", 1, "R", false, 0, "")

	pdf.Ln(4)
	hRule(pdf)
	pdf.Ln(3)

	// ── 3. Seller / Customer columns ─────────────────────────────────────────
	colW := body / 2
	yBefore := pdf.GetY()

	sectionHeader(pdf, font, "ผู้ออกเอกสาร / From", margin, yBefore, colW-5)
	y := pdf.GetY() + 1
	infoRow(pdf, font, margin, y, colW-5, in.SellerName)
	y = pdf.GetY()
	if in.SellerAddress != "" {
		infoRow(pdf, font, margin, y, colW-5, in.SellerAddress)
		y = pdf.GetY()
	}
	if in.SellerTaxID != "" {
		infoRow(pdf, font, margin, y, colW-5, "เลขผู้เสียภาษี: "+in.SellerTaxID)
	}
	yAfterSeller := pdf.GetY()

	xRight := margin + colW
	sectionHeader(pdf, font, "ผู้รับเอกสาร / To", xRight, yBefore, colW)
	y = pdf.GetY() + 1
	infoRow(pdf, font, xRight, y, colW, in.CustomerName)
	y = pdf.GetY()
	if in.CustomerAddress != "" {
		infoRow(pdf, font, xRight, y, colW, in.CustomerAddress)
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

	// ── 4. Invoice table ──────────────────────────────────────────────────────
	if len(in.Invoices) == 0 {
		// No outstanding balance message
		pdf.SetTextColor(100, 116, 139)
		pdf.SetFont(font, "", 10)
		pdf.SetX(margin)
		pdf.CellFormat(body, 12, "ไม่มียอดค้างชำระ / No outstanding balance", "", 1, "C", false, 0, "")
	} else {
		cInv := 30.0
		cIssue := 22.0
		cDue := 22.0
		cAmt := 26.0
		cPaid := 26.0
		cBal := 26.0
		cStat := 18.0
		// total = 170 = body ✓

		// Table header
		pdf.SetFillColor(237, 233, 254)
		pdf.SetTextColor(88, 28, 135)
		pdf.SetFont(font, "", 7.5)
		rowH := 7.0
		pdf.CellFormat(cInv, rowH, "เลขที่ใบแจ้งหนี้", "TB", 0, "L", true, 0, "")
		pdf.CellFormat(cIssue, rowH, "วันที่ออก", "TB", 0, "C", true, 0, "")
		pdf.CellFormat(cDue, rowH, "ครบกำหนด", "TB", 0, "C", true, 0, "")
		pdf.CellFormat(cAmt, rowH, "ยอดรวม", "TB", 0, "R", true, 0, "")
		pdf.CellFormat(cPaid, rowH, "ชำระแล้ว", "TB", 0, "R", true, 0, "")
		pdf.CellFormat(cBal, rowH, "ยอดค้าง", "TB", 0, "R", true, 0, "")
		pdf.CellFormat(cStat, rowH, "สถานะ", "TB", 1, "C", true, 0, "")

		today := time.Now()
		for i, row := range in.Invoices {
			// Row fill based on status
			switch row.Status {
			case "overdue":
				pdf.SetFillColor(254, 242, 242) // red-50
			case "paid":
				pdf.SetFillColor(248, 250, 252) // slate-50
			default:
				if i%2 == 0 {
					pdf.SetFillColor(255, 255, 255)
				} else {
					pdf.SetFillColor(250, 248, 255)
				}
			}

			// Status label and text colour
			statusLabel, statusR, statusG, statusB := statusStyle(row, today)

			pdf.SetTextColor(51, 65, 85)
			if row.Status == "paid" {
				pdf.SetTextColor(148, 163, 184) // slate-400
			}
			pdf.SetFont(font, "", 8.5)
			pdf.CellFormat(cInv, rowH, row.InvoiceNo, "B", 0, "L", true, 0, "")
			pdf.CellFormat(cIssue, rowH, thaiDate(row.IssueDate), "B", 0, "C", true, 0, "")
			pdf.CellFormat(cDue, rowH, thaiDate(row.DueDate), "B", 0, "C", true, 0, "")
			pdf.CellFormat(cAmt, rowH, money(row.Amount), "B", 0, "R", true, 0, "")
			pdf.CellFormat(cPaid, rowH, money(row.Paid), "B", 0, "R", true, 0, "")
			pdf.CellFormat(cBal, rowH, money(row.Balance), "B", 0, "R", true, 0, "")
			pdf.SetTextColor(statusR, statusG, statusB)
			pdf.CellFormat(cStat, rowH, statusLabel, "B", 1, "C", true, 0, "")
		}

		pdf.Ln(4)

		// ── 5. Summary ────────────────────────────────────────────────────────
		sumLabelW := 45.0
		sumValueW := 30.0
		sumX := pageW - margin - sumLabelW - sumValueW

		summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ยอดรวม / Total Amount", money(totalAmount), false, false)
		summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ชำระแล้ว / Total Paid", money(totalPaid), false, false)

		pdf.Ln(1)
		pdf.SetDrawColor(109, 40, 217)
		pdf.SetLineWidth(0.5)
		pdf.Line(sumX, pdf.GetY(), pageW-margin, pdf.GetY())
		pdf.Ln(1)
		summaryRow(pdf, font, sumX, sumLabelW, sumValueW, "ยอดค้างสุทธิ / Net Balance Due", money(netBalance), true, true)
	}

	pdf.Ln(6)
	hRule(pdf)
	pdf.Ln(4)

	// ── 6. Footer ─────────────────────────────────────────────────────────────
	if in.BankName != "" || in.AccountNumber != "" || in.Note != "" {
		pdf.SetFont(font, "", 8)
		pdf.SetTextColor(30, 27, 75)
		pdf.SetX(margin)
		pdf.CellFormat(body/2, 5, "ข้อมูลการชำระเงิน / Payment Details", "", 1, "L", false, 0, "")
		pdf.SetFont(font, "", 9)
		if in.BankName != "" {
			footerLine(pdf, font, "ธนาคาร / Bank:", in.BankName)
		}
		if in.AccountNumber != "" {
			footerLine(pdf, font, "เลขบัญชี / Account:", in.AccountNumber)
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

func statusStyle(row StatementInvoiceRow, today time.Time) (label string, r, g, b int) {
	switch row.Status {
	case "paid":
		return "ชำระแล้ว", 148, 163, 184 // slate-400
	case "overdue":
		return "เกินกำหนด", 220, 38, 38 // red-600
	default:
		return "ค้างชำระ", 217, 119, 6 // amber-600
	}
}
