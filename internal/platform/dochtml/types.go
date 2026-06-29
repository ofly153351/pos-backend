package dochtml

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

// BuildPromptPayQRDataURI generates a PromptPay QR PNG as a data: URI.
// Returns template.URL (safe to use as img src without sanitization).
// Returns "" if promptPayID is empty or invalid.
func BuildPromptPayQRDataURI(promptPayID string, amount float64) template.URL {
	payload := buildPromptPayPayload(promptPayID, amount)
	if payload == "" {
		return ""
	}
	png, err := qrcode.Encode(payload, qrcode.Medium, 256)
	if err != nil {
		return ""
	}
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))
}

var nonDigitRe = regexp.MustCompile(`\D`)

func buildPromptPayPayload(id string, amount float64) string {
	id = normalizePromptPayID(id)
	if id == "" {
		return ""
	}
	var merchantInfo string
	switch len(id) {
	case 13:
		if strings.HasPrefix(id, "0066") {
			merchantInfo = emv("29", emv("00", "A000000677010111")+emv("01", id))
		} else {
			merchantInfo = emv("29", emv("00", "A000000677010111")+emv("02", id))
		}
	case 15:
		merchantInfo = emv("29", emv("00", "A000000677010111")+emv("03", id))
	default:
		return ""
	}
	amountPart := ""
	if amount > 0 {
		amountPart = emv("54", fmt.Sprintf("%.2f", amount))
	}
	raw := "000201" + "010211" + merchantInfo + "5802TH" + "5303764" + amountPart + "6304"
	return raw + strings.ToUpper(fmt.Sprintf("%04X", crc16(raw)))
}

func normalizePromptPayID(s string) string {
	d := nonDigitRe.ReplaceAllString(strings.TrimSpace(s), "")
	switch {
	case len(d) == 13 && strings.HasPrefix(d, "0066"):
		return d
	case len(d) == 11 && strings.HasPrefix(d, "66"):
		return "00" + d
	case len(d) == 10 && strings.HasPrefix(d, "0"):
		return "0066" + d[1:]
	}
	return d
}

func emv(tag, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
}

func crc16(s string) uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range []byte(s) {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// StoreInfo contains store header data used by all document templates.
// BankAccountInfo holds a single store bank account for document rendering.
type BankAccountInfo struct {
	BankName    string
	AccountNo   string
	AccountName string
}

type StoreInfo struct {
	Name         string
	Address      string
	Phone        string
	Fax          string
	Email        string
	Website      string
	TaxID        string
	LogoURL      string
	Branch       string
	PromptPayID  string
	BankAccounts []BankAccountInfo
}

// DocItem is a single line item for HTML rendering.
type DocItem struct {
	Description   string
	DescriptionEn string // optional English description (tax invoice)
	SKU           string // optional product SKU
	Unit          string
	Quantity      float64
	UnitPrice     float64
	DiscountValue float64
	Amount        float64
}

// DocData is a flat representation of a document for HTML rendering.
// It is populated by the document service and passed to RenderDocumentHTML.
type DocData struct {
	Type            string // "INVOICE" | "BILL" | "TAX_INVOICE" | …
	DocumentNo      string
	DocumentNoFull  string
	DocumentDate    time.Time
	DueDate         *time.Time
	ValidUntil      *time.Time
	CustomerName    string
	CustomerAddress string
	CustomerPhone   string
	CustomerTaxID   *string
	CustomerBranch  *string // สาขาผู้ซื้อ (optional)
	StaffName       string
	Items           []DocItem
	Subtotal        float64
	TotalDiscount   float64
	VatRate         float64
	VatAmount       float64
	TotalAmount     float64
	Notes           *string
	// Delivery order fields
	DeliveryDate    *time.Time
	DeliveryAddress string
	DeliveryContact string
	DeliveryPhone   string
	SalesZone       string
	SalespersonName string
	InvoiceRefNo    string
	PORefNo         string
	ShippingFee     float64
	CreditTermDays  int
	PreVatAmount    float64
	QRPaymentURL    template.URL
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
