package sale

import (
	"context"
	"errors"
	"math"
	"strings"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/dochtml"
)

// ErrInvalidSaleDocumentType is returned for an unsupported ?type= value.
var ErrInvalidSaleDocumentType = errors.New("invalid sale document type")

// saleDocumentTypes are the document forms — shared with the Documents module's
// dochtml templates — that can be rendered from a completed sale's data.
var saleDocumentTypes = map[string]bool{
	"INVOICE":        true,
	"TAX_INVOICE":    true,
	"QUOTATION":      true,
	"DELIVERY_ORDER": true,
	"BILL":           true,
	"CREDIT_NOTE":    true,
	"DEBIT_NOTE":     true,
}

// GenerateSaleDocumentHTML renders a sale through the EXISTING document templates
// (internal/platform/dochtml) — the very forms the Documents menu prints — so a
// full tax invoice / delivery note / quotation issued from Sales History is
// identical to one issued from the Documents module. Rendering only: no sale,
// price, VAT, promotion, stock or payment data is mutated.
func (s Service) GenerateSaleDocumentHTML(ctx context.Context, actor auth.Claims, storeID, saleID, docType string) ([]byte, error) {
	docType = strings.ToUpper(strings.TrimSpace(docType))
	if docType == "" {
		docType = "TAX_INVOICE"
	}
	if !saleDocumentTypes[docType] {
		return nil, ErrInvalidSaleDocumentType
	}

	saleRecord, err := s.GetByID(ctx, actor, storeID, saleID)
	if err != nil {
		return nil, err
	}

	// VAT is broken out on the document only when the store charges it exclusively —
	// mirroring GenerateReceiptHTML — so a rendered tax invoice matches the receipt.
	sv := s.loadSettings(ctx, storeID)
	vatRate, vatAmount := documentVAT(saleRecord, sv)

	docData := buildSaleDocData(saleRecord, docType, vatRate, vatAmount)
	if strings.TrimSpace(saleRecord.StorePromptPayID) != "" {
		docData.QRPaymentURL = dochtml.BuildPromptPayQRDataURI(saleRecord.StorePromptPayID, saleRecord.TotalAmount)
	}

	bankRows := s.repo.GetStoreBankAccounts(ctx, storeID)
	bankAccounts := make([]dochtml.BankAccountInfo, len(bankRows))
	for i, r := range bankRows {
		bankAccounts[i] = dochtml.BankAccountInfo{
			BankName:    r.BankName,
			AccountNo:   r.AccountNo,
			AccountName: r.AccountName,
		}
	}

	html, err := dochtml.RenderDocumentHTML(docData, dochtml.StoreInfo{
		Name:         saleRecord.StoreName,
		Address:      saleRecord.StoreAddress,
		Phone:        saleRecord.StorePhone,
		TaxID:        saleRecord.StoreTaxID,
		LogoURL:      strings.TrimSpace(saleRecord.StoreLogoURL),
		PromptPayID:  strings.TrimSpace(saleRecord.StorePromptPayID),
		BankAccounts: bankAccounts,
	})
	if err != nil {
		return nil, err
	}
	return []byte(html), nil
}

// StoreBankAccountInfos exposes the store's bank accounts as dochtml rows so
// sibling modules (e.g. credit-sales billing notices) can render the shared
// document templates with the same payment block. Read-only.
func (s Service) StoreBankAccountInfos(ctx context.Context, storeID string) []dochtml.BankAccountInfo {
	rows := s.repo.GetStoreBankAccounts(ctx, storeID)
	out := make([]dochtml.BankAccountInfo, len(rows))
	for i, r := range rows {
		out[i] = dochtml.BankAccountInfo{
			BankName:    r.BankName,
			AccountNo:   r.AccountNo,
			AccountName: r.AccountName,
		}
	}
	return out
}

// documentVAT returns the VAT rate/amount a customer-facing document should display.
// It mirrors the receipt (GenerateReceiptHTML): VAT is only broken out for stores that
// charge it exclusively. Inclusive / no-VAT stores show no VAT line — the tax it
// already embeds keeps the grand total equal to the amount the customer paid.
func documentVAT(s Sale, sv ReceiptSettingsView) (rate, amount float64) {
	if sv.TaxMode == "exclusive" && s.VATPercent > 0 {
		return s.VATPercent, roundMoney(s.VATAmount)
	}
	return 0, 0
}

// buildSaleDocData maps a Sale onto the dochtml.DocData contract, mirroring the
// document module's toDocData so the shared templates render consistently.
// vatRate/vatAmount are the display values resolved by documentVAT (TaxMode-aware).
func buildSaleDocData(s Sale, docType string, vatRate, vatAmount float64) dochtml.DocData {
	items := make([]dochtml.DocItem, len(s.Items))
	for i, it := range s.Items {
		items[i] = dochtml.DocItem{
			Description:   it.ProductName,
			SKU:           it.SKU,
			Unit:          it.UnitType,
			Quantity:      float64(it.Quantity),
			UnitPrice:     it.UnitPrice,
			DiscountValue: it.LineDiscountTotal,
			Amount:        it.LineTotal,
		}
	}

	// DiscountAmount is ALREADY the combined item + bill discount (set by the sale
	// repository). Do NOT add BillDiscountAmount again — that double-counted the bill
	// discount and inflated the document's discount line vs the receipt.
	totalDiscount := s.DiscountAmount
	// Net of VAT — robust for both VAT-included and VAT-excluded sales.
	preVat := math.Round((s.TotalAmount-vatAmount)*100) / 100

	docDate := s.SoldAt
	if docDate.IsZero() {
		docDate = s.CreatedAt
	}

	var notes *string
	if strings.TrimSpace(s.Note) != "" {
		n := s.Note
		notes = &n
	}
	var customerTaxID *string
	if strings.TrimSpace(s.CustomerTaxID) != "" {
		t := s.CustomerTaxID
		customerTaxID = &t
	}
	var customerBranch *string
	if strings.TrimSpace(s.CustomerBranch) != "" {
		b := s.CustomerBranch
		customerBranch = &b
	}

	return dochtml.DocData{
		Type:           docType,
		DocumentNo:     s.SaleNumber,
		DocumentNoFull: s.SaleNumber,
		DocumentDate:   docDate,
		CustomerName:   s.CustomerName,
		CustomerPhone:  s.CustomerPhone,
		CustomerTaxID:  customerTaxID,
		CustomerBranch: customerBranch,
		StaffName:      s.CashierName,
		Items:          items,
		Subtotal:       s.SubtotalAmount,
		TotalDiscount:  totalDiscount,
		VatRate:        vatRate,
		VatAmount:      vatAmount,
		TotalAmount:    s.TotalAmount,
		PreVatAmount:   preVat,
		Notes:          notes,
	}
}
