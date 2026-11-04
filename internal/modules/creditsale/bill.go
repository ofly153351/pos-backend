package creditsale

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/dochtml"
)

// BillHTML renders a single credit sale as a ใบวางบิล (BILL) through the shared
// unified document template — the same form the Documents module and Sales History
// print. Rendering only: no sale, price, VAT, stock or payment data is mutated.
//
// The credit sale wraps a real product-backed sale (cs.SaleID); its store header
// (name/address/phone/tax id/logo/PromptPay) and bank accounts are pulled from that
// sale so the billing notice is byte-for-byte consistent with a sale-issued document.
func (s Service) BillHTML(ctx context.Context, actor auth.Claims, storeID, creditSaleID string) (string, error) {
	cs, err := s.repo.Get(ctx, storeID, creditSaleID)
	if err != nil {
		return "", err
	}

	store := dochtml.StoreInfo{}
	if strings.TrimSpace(cs.SaleID) != "" {
		if underlying, sErr := s.saleService.GetByID(ctx, actor, storeID, cs.SaleID); sErr == nil {
			store = dochtml.StoreInfo{
				Name:         underlying.StoreName,
				Address:      underlying.StoreAddress,
				Phone:        underlying.StorePhone,
				TaxID:        underlying.StoreTaxID,
				LogoURL:      strings.TrimSpace(underlying.StoreLogoURL),
				PromptPayID:  strings.TrimSpace(underlying.StorePromptPayID),
				BankAccounts: s.saleService.StoreBankAccountInfos(ctx, storeID),
			}
		}
	}

	in := mapCreditSaleToSaleInput(cs)

	doc := dochtml.BuildCreditSaleBillDocData(in, store)
	// BILL shows one row per receivable; product lines belong to the sale.
	doc.Items = nil

	// B08: one source for every amount on the bill — the customer's open
	// credit receivables. Rows show original / paid / remaining, the totals box
	// is the sum of the same rows (pre-tax, VAT from the stored sales), and the
	// QR asks for the remaining balance (owner rule: bills request the balance).
	statement, err := s.repo.StatementContext(ctx, storeID, creditSaleID)
	if err != nil {
		return "", err
	}
	open := make([]CreditSale, 0, len(statement.Sales))
	saleIDs := make([]string, 0, len(statement.Sales))
	for _, sale := range statement.Sales {
		if sale.Type != typeCredit || sale.Status == StatusCancelled || sale.RemainingAmount <= 0 {
			continue
		}
		open = append(open, sale)
		saleIDs = append(saleIDs, sale.SaleID)
	}
	vatBySale, err := s.repo.SaleVATAmounts(ctx, storeID, saleIDs)
	if err != nil {
		return "", err
	}
	doc.BillSourceHeading = "เอกสารขายเชื่อ (Credit sale)"
	doc.BillRowPrefix = "เลขที่"
	doc.BillShowBalance = true
	doc.BillRows = make([]dochtml.BillRow, 0, len(open))
	var total, vat, remaining float64
	for _, sale := range open {
		issue := parseTimestamp(sale.CreatedAt)
		if issue.IsZero() {
			issue = time.Now()
		}
		var due *time.Time
		if parsed := parseDateOnly(sale.DueDate); !parsed.IsZero() {
			due = &parsed
		}
		doc.BillRows = append(doc.BillRows, dochtml.BillRow{
			DocumentNo: sale.DocumentNumber,
			IssueDate:  issue,
			DueDate:    due,
			Amount:     sale.TotalAmount,
			Paid:       sale.PaidAmount,
			Remaining:  sale.RemainingAmount,
		})
		total += sale.TotalAmount
		vat += vatBySale[sale.SaleID]
		remaining += sale.RemainingAmount
	}
	total, vat, remaining = roundMoney(total), roundMoney(vat), roundMoney(remaining)
	doc.TotalAmount = total
	doc.VatAmount = vat
	doc.Subtotal = roundMoney(total - vat)
	doc.PreVatAmount = doc.Subtotal
	doc.TotalDiscount = 0
	if total > 0 && vat > 0 {
		doc.VatRate = math.Round(vat/(total-vat)*10000) / 100
	} else {
		doc.VatRate = 0
	}
	doc.QRPaymentURL = ""
	if store.PromptPayID != "" && remaining > 0 {
		doc.QRPaymentURL = dochtml.BuildPromptPayQRDataURI(store.PromptPayID, remaining)
	}
	return dochtml.RenderUnifiedDocumentHTML(doc, store)
}

// mapCreditSaleToSaleInput maps the real CreditSale model onto the dochtml SaleInput
// contract. Credit sales are minted with VAT disabled (Create sets VATIncluded=false),
// so VatRate is 0 and the inclusive→exclusive converter is a pass-through — prices are
// rendered exactly as stored. A per-line discount is back-derived from the stored line
// total so the rendered amount always equals the recorded total (never recomputed).
func mapCreditSaleToSaleInput(cs CreditSale) dochtml.SaleInput {
	lines := make([]dochtml.SaleLineInput, 0, len(cs.Items))
	for _, it := range cs.Items {
		gross := float64(it.Quantity) * it.Price
		lineDiscount := gross - it.Total
		if lineDiscount < 0 {
			lineDiscount = 0
		}
		lines = append(lines, dochtml.SaleLineInput{
			Description:         it.ProductName,
			Unit:                it.Unit,
			Quantity:            float64(it.Quantity),
			UnitPriceInclVat:    it.Price,
			LineDiscountInclVat: lineDiscount,
		})
	}

	var due *time.Time
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(cs.DueDate)); err == nil {
		due = &t
	}

	issue := parseTimestamp(cs.CreatedAt)
	if issue.IsZero() {
		issue = time.Now()
	}

	note := buildBillNote(cs)

	return dochtml.SaleInput{
		DocumentNo:     cs.DocumentNumber,
		DocumentNoFull: cs.DocumentNumber,
		IssueDate:      issue,
		DueDate:        due,
		Customer: dochtml.CustomerInfo{
			Name:  cs.CustomerName,
			Phone: cs.CustomerPhone,
		},
		Lines:   lines,
		VatRate: 0,
		Notes:   &note,
	}
}

// buildBillNote folds the user note together with the receivable's paid/outstanding
// summary so the billing notice still communicates the balance due — the unified
// template has no dedicated paid/remaining row.
func buildBillNote(cs CreditSale) string {
	var b strings.Builder
	if n := strings.TrimSpace(cs.Note); n != "" {
		b.WriteString(n)
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "ยอดรวม %.2f · ชำระแล้ว %.2f · คงเหลือ %.2f บาท",
		cs.TotalAmount, cs.PaidAmount, cs.RemainingAmount)
	return b.String()
}
