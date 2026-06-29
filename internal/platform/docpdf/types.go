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

	// Document identity + copy context (per Thai Revenue copy/original rules).
	DocTitleTH    string // e.g. "ใบส่งของ / ใบกำกับภาษี" (empty → falls back to Invoice)
	DocTitleEN    string // e.g. "Delivery Note / Tax Invoice"
	BadgeText     string // e.g. "ต้นฉบับ (Original)" / "สำเนา (Copy)"
	PurposeText   string // e.g. "(สำหรับลูกค้า)"
	ShowSignature bool   // draw the goods-received signature block on this copy

	Items []InvoicePDFItem
	// Stored, authoritative totals — taken verbatim from the document (toDocData),
	// never recomputed here. This is what guarantees the PDF matches the receipt /
	// HTML document / stored sale exactly.
	Subtotal      float64 // Σ pre-discount line amounts
	TotalDiscount float64 // Σ line discounts + bill discount
	PreVatAmount  float64 // taxable base = Subtotal - TotalDiscount
	VATRate       float64 // 0 = no VAT line
	VATAmount     float64
	TotalAmount   float64 // grand total (stored, already rounded)

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
	// LineDiscount is the per-line discount value (stored). LineAmount is the
	// post-discount line total (stored doc item Amount). Both come verbatim from
	// the document so printed rows reconcile to the Subtotal box.
	LineDiscount float64
	LineAmount   float64
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
