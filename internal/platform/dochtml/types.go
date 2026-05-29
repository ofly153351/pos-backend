package dochtml

import "time"

// StoreInfo contains store header data used by all document templates.
type StoreInfo struct {
	Name    string
	Address string
	Phone   string
	Fax     string
	Email   string
	Website string
	TaxID   string
	LogoURL string
}

// DocItem is a single line item for HTML rendering.
type DocItem struct {
	Description   string
	Unit          string
	Quantity      float64
	UnitPrice     float64
	DiscountValue float64
	Amount        float64
}

// DocData is a flat representation of a document for HTML rendering.
// It is populated by the document service and passed to RenderDocumentHTML.
type DocData struct {
	Type           string // "INVOICE" | "BILL" | "TAX_INVOICE" | …
	DocumentNo     string
	DocumentNoFull string
	DocumentDate   time.Time
	DueDate        *time.Time
	CustomerName   string
	CustomerAddress string
	CustomerPhone  string
	CustomerTaxID  *string
	StaffName      string
	Items          []DocItem
	Subtotal       float64
	TotalDiscount  float64
	VatRate        float64
	VatAmount      float64
	TotalAmount    float64
	Notes          *string
}

// WHTCertData holds all data needed to render a WHT certificate (ภ.ง.ด.3/53).
type WHTCertData struct {
	// Payer = store
	PayerName    string
	PayerAddress string
	PayerTaxID   string

	// Payee = customer
	PayeeName    string
	PayeeAddress string
	PayeeTaxID   string

	ReceiverType string // "individual" | "company"
	FormNo       string // "ภ.ง.ด.3" | "ภ.ง.ด.53"

	DocumentNo  string
	PaymentDate time.Time

	IncomeType  string
	IncomeDesc  string
	GrossAmount float64
	WHTRate     float64
	WHTAmount   float64
	NetAmount   float64
}

// pageData holds data for a single printed page.
type pageData struct {
	Doc        DocData
	Store      StoreInfo
	Items      []DocItem  // items for this page only
	FillerRows []struct{} // empty filler rows (last page only)
	ItemOffset int        // global index of first item on this page
	PageNo     int
	TotalPages int
	IsFirst    bool
	IsLast     bool
}

// renderData is the template data envelope for paginated invoice (unexported).
type renderData struct {
	Pages []pageData
}

// billRenderData is the template data envelope for bill (no pagination).
type billRenderData struct {
	Doc        DocData
	Store      StoreInfo
	FillerRows []struct{}
}
