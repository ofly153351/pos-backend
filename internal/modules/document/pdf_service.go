package document

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/docpdf"
)

// GenerateDocumentPDF builds the correct PDF for a document based on its type.
// Bill (TypeBill) uses RenderBillPDF; all other types use RenderInvoicePDF.
func (s Service) GenerateDocumentPDF(ctx context.Context, actor auth.Claims, storeID, docID string, opts InvoicePDFOptions) ([]byte, string, error) {
	doc, err := s.GetDocument(ctx, actor, storeID, docID)
	if err != nil {
		return nil, "", err
	}

	if doc.Type == TypeBill {
		return s.generateBillPDF(doc, opts)
	}
	return s.generateInvoicePDF(doc, opts)
}

// GenerateInvoicePDF is an alias kept for backward compatibility.
func (s Service) GenerateInvoicePDF(ctx context.Context, actor auth.Claims, storeID, docID string, opts InvoicePDFOptions) ([]byte, string, error) {
	return s.GenerateDocumentPDF(ctx, actor, storeID, docID, opts)
}

func (s Service) generateInvoicePDF(doc *Document, opts InvoicePDFOptions) ([]byte, string, error) {
	dueDate := doc.DocumentDate
	if doc.DueDate != nil {
		dueDate = *doc.DueDate
	}

	// Use stored customer address/TaxID as fallback (§86/4 compliance)
	custAddr := opts.CustomerAddress
	if custAddr == "" {
		custAddr = doc.CustomerAddress
	}
	custTaxID := opts.CustomerTaxID
	if custTaxID == "" && doc.CustomerTaxID != nil {
		custTaxID = *doc.CustomerTaxID
	}
	defaultUnit := opts.DefaultUnit
	if defaultUnit == "" {
		defaultUnit = "ชิ้น"
	}

	logoBytes, logoExt := fetchLogo(doc.StoreLogoURL)
	// Canonical numbers — identical to what the HTML document / receipt render.
	// The PDF prints these verbatim; opts.DiscountPercent is intentionally ignored
	// (it was the source of the discount-omitted / VAT-on-pre-discount bug).
	dd := toDocData(doc)
	in := docpdf.InvoicePDFInput{
		SellerName:      doc.StoreName,
		SellerAddress:   doc.StoreAddress,
		SellerTaxID:     doc.StoreTaxID,
		SellerPhone:     doc.StorePhone,
		SellerLogoBytes: logoBytes,
		SellerLogoExt:   logoExt,

		CustomerName:    doc.CustomerName,
		CustomerAddress: custAddr,
		CustomerTaxID:   custTaxID,
		CreditTerm:      opts.CreditTerm,

		InvoiceNo:   doc.DocumentNoFull,
		IssueDate:   doc.DocumentDate,
		DueDate:     dueDate,
		ReferenceDO: opts.ReferenceDO,

		Subtotal:      dd.Subtotal,
		TotalDiscount: dd.TotalDiscount,
		PreVatAmount:  dd.PreVatAmount,
		VATRate:       dd.VatRate,
		VATAmount:     dd.VatAmount,
		TotalAmount:   dd.TotalAmount,

		BankName:      opts.BankName,
		AccountNumber: opts.AccountNumber,
		PromptPay:     opts.PromptPay,
		Note:          noteStr(doc.Notes),
	}
	for _, it := range dd.Items {
		unit := it.Unit
		if unit == "" {
			unit = defaultUnit
		}
		in.Items = append(in.Items, docpdf.InvoicePDFItem{
			Description:  it.Description,
			Quantity:     it.Quantity,
			Unit:         unit,
			UnitPrice:    it.UnitPrice,
			LineDiscount: it.DiscountValue,
			LineAmount:   it.Amount,
		})
	}

	pdfBytes, err := docpdf.RenderInvoicePDF(in)
	if err != nil {
		return nil, "", err
	}
	outPath := pdfOutputPath("INVOICE", doc.DocumentNo)
	if err := writePDF(outPath, pdfBytes); err != nil {
		return nil, "", err
	}
	return pdfBytes, outPath, nil
}

func (s Service) generateBillPDF(doc *Document, opts InvoicePDFOptions) ([]byte, string, error) {
	custAddr := doc.CustomerAddress
	if opts.CustomerAddress != "" {
		custAddr = opts.CustomerAddress
	}
	custTaxID := ""
	if doc.CustomerTaxID != nil {
		custTaxID = *doc.CustomerTaxID
	}
	if opts.CustomerTaxID != "" {
		custTaxID = opts.CustomerTaxID
	}

	billLogoBytes, billLogoExt := fetchLogo(doc.StoreLogoURL)
	// Same canonical numbers as the HTML document / receipt — printed verbatim.
	dd := toDocData(doc)
	in := docpdf.BillPDFInput{
		SellerName:      doc.StoreName,
		SellerAddress:   doc.StoreAddress,
		SellerTaxID:     doc.StoreTaxID,
		SellerPhone:     doc.StorePhone,
		SellerLogoBytes: billLogoBytes,
		SellerLogoExt:   billLogoExt,

		CustomerName:    doc.CustomerName,
		CustomerAddress: custAddr,
		CustomerTaxID:   custTaxID,

		BillNo:    doc.DocumentNoFull,
		IssueDate: doc.DocumentDate,
		DueDate:   doc.DueDate,

		Subtotal:      dd.Subtotal,
		TotalDiscount: dd.TotalDiscount,
		PreVatAmount:  dd.PreVatAmount,
		VATRate:       dd.VatRate,
		VATAmount:     dd.VatAmount,
		TotalAmount:   dd.TotalAmount,
		Note:          noteStr(doc.Notes),
	}
	defaultUnit := opts.DefaultUnit
	if defaultUnit == "" {
		defaultUnit = "ชิ้น"
	}
	for _, it := range dd.Items {
		unit := it.Unit
		if unit == "" {
			unit = defaultUnit
		}
		in.Items = append(in.Items, docpdf.InvoicePDFItem{
			Description:  it.Description,
			Quantity:     it.Quantity,
			Unit:         unit,
			UnitPrice:    it.UnitPrice,
			LineDiscount: it.DiscountValue,
			LineAmount:   it.Amount,
		})
	}

	pdfBytes, err := docpdf.RenderBillPDF(in)
	if err != nil {
		return nil, "", err
	}
	outPath := pdfOutputPath("BILL", doc.DocumentNo)
	if err := writePDF(outPath, pdfBytes); err != nil {
		return nil, "", err
	}
	return pdfBytes, outPath, nil
}

// GenerateStatementPDF builds a Statement PDF for a customer over a date range.
func (s Service) GenerateStatementPDF(ctx context.Context, actor auth.Claims, storeID, customerID string, opts StatementPDFOptions) ([]byte, string, error) {

	// Fetch store info
	var store struct {
		Name    string
		Address string
		TaxID   string
	}
	_ = s.db.Raw(
		"SELECT name, COALESCE(address,'') AS address, COALESCE(tax_id,'') AS tax_id FROM stores WHERE id = ?",
		storeID,
	).Scan(&store)

	// Fetch customer
	var cust struct {
		FullName string
		Address  string
	}
	_ = s.db.Raw(
		"SELECT full_name, COALESCE(address,'') AS address FROM customers WHERE id = ? AND store_id = ?",
		customerID, storeID,
	).Scan(&cust)

	// Fetch documents for customer in period that are not fully paid
	var docs []Document
	err := s.db.
		Preload("Items").
		Where("store_id = ? AND customer_id = ? AND document_date BETWEEN ? AND ?",
			storeID, customerID, opts.PeriodStart, opts.PeriodEnd).
		Where("payment_status != ?", string(PaymentPaid)).
		Order("document_date ASC").
		Find(&docs).Error
	if err != nil {
		return nil, "", err
	}

	// Seq number for this statement
	seq, _ := s.repo.NextSeq(storeID, TypeBill) // reuse bill sequence for statements
	now := time.Now()
	buddhistYear := now.Year() + 543
	stmtNo := fmt.Sprintf("STMT/%d/%02d/%04d", buddhistYear, int(now.Month()), seq)

	today := now.Truncate(24 * time.Hour)
	var rows []docpdf.StatementInvoiceRow
	for _, d := range docs {
		balance := d.TotalAmount // simplified: no partial payment tracking in doc module
		due := d.DocumentDate
		if d.DueDate != nil {
			due = *d.DueDate
		}
		status := "outstanding"
		if due.Before(today) && balance > 0 {
			status = "overdue"
		}
		rows = append(rows, docpdf.StatementInvoiceRow{
			InvoiceNo: d.DocumentNoFull,
			IssueDate: d.DocumentDate,
			DueDate:   due,
			Amount:    d.TotalAmount,
			Paid:      0,
			Balance:   balance,
			Status:    status,
		})
	}

	in := docpdf.StatementPDFInput{
		SellerName:    store.Name,
		SellerAddress: store.Address,
		SellerTaxID:   store.TaxID,

		CustomerName:    cust.FullName,
		CustomerAddress: cust.Address,

		StatementNo: stmtNo,
		IssueDate:   now,
		PeriodStart: opts.PeriodStart,
		PeriodEnd:   opts.PeriodEnd,

		Invoices: rows,

		BankName:      opts.BankName,
		AccountNumber: opts.AccountNumber,
		Note:          opts.Note,
	}

	pdfBytes, err := docpdf.RenderStatementPDF(in)
	if err != nil {
		return nil, "", err
	}

	safeStmt := strings.ReplaceAll(stmtNo, "/", "-")
	outPath := pdfOutputPath("STATEMENT", safeStmt)
	if err := writePDF(outPath, pdfBytes); err != nil {
		return nil, "", err
	}
	return pdfBytes, outPath, nil
}

// ── Options ───────────────────────────────────────────────────────────────────

type InvoicePDFOptions struct {
	CustomerAddress string
	CustomerTaxID   string
	CreditTerm      int
	ReferenceDO     string
	DiscountPercent float64
	DefaultUnit     string
	BankName        string
	AccountNumber   string
	PromptPay       string
}

type StatementPDFOptions struct {
	PeriodStart   time.Time
	PeriodEnd     time.Time
	BankName      string
	AccountNumber string
	Note          string
}

// ── Private helpers ───────────────────────────────────────────────────────────

func pdfOutputPath(docType, docNo string) string {
	dir := "outputs"
	_ = os.MkdirAll(dir, 0o755)
	name := fmt.Sprintf("%s_%s.pdf", docType, strings.ReplaceAll(docNo, "/", "-"))
	return filepath.Join(dir, name)
}

func writePDF(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func noteStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// fetchLogo downloads logo bytes from a URL. Returns nil bytes on any error.
func fetchLogo(rawURL string) (data []byte, ext string) {
	if rawURL == "" {
		return nil, ""
	}
	resp, err := http.Get(rawURL) //nolint:noctx
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, ""
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, ""
	}
	ct := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "png"):
		ext = "png"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		ext = "jpeg"
	default:
		// infer from URL
		lower := strings.ToLower(rawURL)
		if strings.HasSuffix(lower, ".png") {
			ext = "png"
		} else {
			ext = "jpeg"
		}
	}
	return data, ext
}
