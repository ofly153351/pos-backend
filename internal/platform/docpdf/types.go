package docpdf

import "time"

// ── Invoice PDF ───────────────────────────────────────────────────────────────

type InvoicePDFInput struct {
	SellerName      string
	SellerAddress   string
	SellerTaxID     string
	SellerPhone     string
	SellerLogoBytes []byte // optional — fetched from URL by caller
	SellerLogoExt   string // "png" | "jpeg"

	CustomerName    string
	CustomerAddress string
	CustomerTaxID   string
	CreditTerm      int // days

	InvoiceNo   string
	IssueDate   time.Time
	DueDate     time.Time
	ReferenceDO string // optional

	Items           []InvoicePDFItem
	DiscountPercent float64 // 0 = no discount
	VATRegistered   bool

	BankName      string
	AccountNumber string
	PromptPay     string // optional
	Note          string
}

type InvoicePDFItem struct {
	Description string
	Quantity    float64
	Unit        string
	UnitPrice   float64
}

// ── Statement PDF ─────────────────────────────────────────────────────────────

type StatementPDFInput struct {
	SellerName    string
	SellerAddress string
	SellerTaxID   string

	CustomerName    string
	CustomerAddress string

	StatementNo string
	IssueDate   time.Time
	PeriodStart time.Time
	PeriodEnd   time.Time

	Invoices []StatementInvoiceRow

	BankName      string
	AccountNumber string
	Note          string
}

type StatementInvoiceRow struct {
	InvoiceNo string
	IssueDate time.Time
	DueDate   time.Time
	Amount    float64
	Paid      float64
	Balance   float64
	// "overdue" | "outstanding" | "paid"
	Status string
}
