package receipt_settings

import (
	"context"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/store"
	"pos-backend/internal/platform/activitycapture"
	"pos-backend/internal/platform/dochtml"
	"pos-backend/internal/platform/receipthtml"
)

type Service struct {
	repo      Repository
	storeRepo store.Repository
}

func NewService(repo Repository, storeRepo store.Repository) Service {
	return Service{repo: repo, storeRepo: storeRepo}
}

func (s Service) Get(ctx context.Context, actor auth.Claims, storeID string) (ReceiptSettings, error) {

	settings, err := s.repo.GetByStoreID(ctx, storeID)
	if err == ErrNotFound {
		return s.createDefaults(ctx, storeID)
	}
	if err != nil {
		return ReceiptSettings{}, err
	}
	normalizeDocumentDesign(&settings.DocumentDesign)
	return settings, nil
}

func normalizeDocumentDesign(design *DocumentDesign) {
	if design.FontFamily == "" {
		design.FontFamily = "Sarabun"
	}
	if design.Spacing == "" {
		design.Spacing = "standard"
	}
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID string, input UpdateReceiptSettingsRequest) (ReceiptSettings, error) {

	if input.DocumentTemplateKey != nil && !validDocumentTemplateKey(*input.DocumentTemplateKey) {
		return ReceiptSettings{}, ErrInvalidInput
	}
	if input.DocumentDesign != nil {
		d := *input.DocumentDesign
		if d.FontSize < 8 || d.FontSize > 16 || d.MarginMM < 5 || d.MarginMM > 30 || (d.Spacing != "compact" && d.Spacing != "standard" && d.Spacing != "comfortable") || (d.FontFamily != "Sarabun" && d.FontFamily != "Tahoma" && d.FontFamily != "Arial") || (d.Alignment != "left" && d.Alignment != "center" && d.Alignment != "right") || !validHexColor(d.PrimaryColor) {
			return ReceiptSettings{}, ErrInvalidInput
		}
	}
	current, err := s.repo.GetByStoreID(ctx, storeID)
	if err == ErrNotFound {
		current, err = s.createDefaults(ctx, storeID)
		if err != nil {
			return ReceiptSettings{}, err
		}
	} else if err != nil {
		return ReceiptSettings{}, err
	}

	beforeSnapshot := activitySnapshot(current)
	applyUpdate(&current, input)
	updated, err := s.repo.Upsert(ctx, current)
	if err != nil {
		return ReceiptSettings{}, err
	}
	activitycapture.Record(ctx, "receipt_settings", beforeSnapshot, activitySnapshot(updated))
	return updated, nil
}

func (s Service) createDefaults(ctx context.Context, storeID string) (ReceiptSettings, error) {
	now := time.Now().UTC()
	defaults := ReceiptSettings{
		ID:                  newID(),
		StoreID:             storeID,
		DocumentTemplateKey: "standard",
		DocumentDesign:      DocumentDesign{PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12, ShowLogo: true, ShowStoreInfo: true, Spacing: "standard", FontFamily: "Sarabun"},
		TemplateKey:         "modern_classic",
		PaperSize:           "80mm",
		PaperLength:         "auto",
		TaxMode:             "exclusive",
		VatRate:             7.00,
		TaxLabel:            "ภาษีมูลค่าเพิ่ม (VAT 7%)",
		ShowLogo:            true,
		LogoPosition:        "top_center",
		ShowStoreName:       true,
		ShowAddress:         true,
		ShowPhone:           true,
		ShowTaxId:           true,
		FooterText:          "ขอบคุณที่ใช้บริการ\nสินค้าที่ซื้อแล้วไม่รับเปลี่ยนคืนทุกกรณี\nพบปัญหาสินค้า กรุณาติดต่อภายใน 7 วัน",
		PaymentChannels:     defaultChannels(),
		PrinterType:         "thermal",
		PrinterName:         "",
		AutoPrint:           false,
		Copies:              1,
		ShowQr:              true,
		QrSize:              "medium",
		ShowCustomerDisplay: false,
		ShowProductImages:   false,
		DateFormat:          "DD/MM/YYYY",
		TimeFormat:          "24h",
		CurrencyPosition:    "before",
		RoundAmount:         true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	result, err := s.repo.Insert(ctx, defaults)
	if err != nil {
		// Concurrent insert — just fetch whatever is there
		return s.repo.GetByStoreID(ctx, storeID)
	}
	return result, nil
}

// defaultChannels — the canonical payment channels (keys mirror lib/payment-method.ts
// on the frontend). Credit & debit cards are one combined "card" channel.
func defaultChannels() PaymentChannels {
	return PaymentChannels{
		{Key: "cash", Enabled: true},
		{Key: "bank_transfer", Enabled: true},
		{Key: "promptpay", Enabled: true},
		{Key: "card", Enabled: false},
		{Key: "truemoney", Enabled: false},
		{Key: "shopeepay", Enabled: false},
	}
}

func validDocumentTemplateKey(key string) bool {
	return key == "standard" || key == "classic" || key == "template3"
}

func (s Service) PreviewDocumentHTML(ctx context.Context, actor auth.Claims, storeID string, input UpdateReceiptSettingsRequest) ([]byte, error) {
	// Preview reads the current store identity and design only. Customer, items
	// and document fields are fixture-only; missing settings must not create rows.
	settings, err := s.repo.GetByStoreID(ctx, storeID)
	if err == ErrNotFound {
		settings = ReceiptSettings{
			DocumentTemplateKey: "standard",
			DocumentDesign:      DocumentDesign{PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12, ShowLogo: true, ShowStoreInfo: true, Spacing: "standard", FontFamily: "Sarabun"},
		}
	} else if err != nil {
		return nil, err
	}
	normalizeDocumentDesign(&settings.DocumentDesign)
	applyUpdate(&settings, input)
	if !validDocumentTemplateKey(settings.DocumentTemplateKey) || !validHexColor(settings.DocumentDesign.PrimaryColor) || settings.DocumentDesign.FontSize < 8 || settings.DocumentDesign.FontSize > 16 || (settings.DocumentDesign.Spacing != "compact" && settings.DocumentDesign.Spacing != "standard" && settings.DocumentDesign.Spacing != "comfortable") || (settings.DocumentDesign.FontFamily != "Sarabun" && settings.DocumentDesign.FontFamily != "Tahoma" && settings.DocumentDesign.FontFamily != "Arial") || (settings.DocumentDesign.Alignment != "left" && settings.DocumentDesign.Alignment != "center" && settings.DocumentDesign.Alignment != "right") || settings.DocumentDesign.MarginMM < 5 || settings.DocumentDesign.MarginMM > 30 {
		return nil, ErrInvalidInput
	}
	storeRecord, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}
	var contact []string
	if storeRecord.Email != "" {
		contact = append(contact, "Email: "+storeRecord.Email)
	}
	if storeRecord.Fax != "" {
		contact = append(contact, "Fax: "+storeRecord.Fax)
	}
	if storeRecord.Website != "" {
		contact = append(contact, "Web: "+storeRecord.Website)
	}
	storeInfo := dochtml.StoreInfo{
		Name: storeRecord.Name, Address: storeRecord.Address, Phone: storeRecord.Phone,
		Fax: storeRecord.Fax, Email: storeRecord.Email, Website: storeRecord.Website,
		TaxID: storeRecord.TaxID, LogoURL: previewStoreLogo(ctx, storeRecord.LogoURL),
		PreviewContact: strings.Join(contact, " · "),
	}
	doc := documentPreviewFixture()
	var html string
	if settings.DocumentTemplateKey == "classic" {
		html, err = dochtml.RenderReferenceDocumentCopies(doc, storeInfo, 0)
	} else if settings.DocumentTemplateKey == "template3" {
		html, err = dochtml.RenderTemplate3DocumentCopies(doc, storeInfo, 0)
	} else {
		html, err = dochtml.RenderUnifiedDocumentCopies(doc, storeInfo, 0)
	}
	if err != nil {
		return nil, err
	}
	html, err = dochtml.ApplyDocumentTemplateDesign(html, settings.DocumentTemplateKey, dochtml.DocumentDesign(settings.DocumentDesign))
	if err != nil {
		return nil, err
	}
	return []byte(html), nil
}

// PreviewHTML renders a preview receipt using the provided settings override + real store info.
func (s Service) PreviewHTML(ctx context.Context, actor auth.Claims, storeID string, input UpdateReceiptSettingsRequest) ([]byte, error) {

	// Load the current saved settings as the base, then apply overrides.
	base, err := s.repo.GetByStoreID(ctx, storeID)
	if err == ErrNotFound {
		base, err = s.createDefaults(ctx, storeID)
	}
	if err != nil {
		return nil, err
	}
	applyUpdate(&base, input)

	// Load real store info for preview header.
	storeRecord, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}

	cfg := receipthtml.Config{
		ShowLogo:      base.ShowLogo,
		LogoPosition:  base.LogoPosition,
		LogoURL:       storeRecord.LogoURL,
		ShowStoreName: base.ShowStoreName,
		ShowAddress:   base.ShowAddress,
		ShowPhone:     base.ShowPhone,
		ShowTaxId:     base.ShowTaxId,
		TaxMode:       base.TaxMode,
		VatRate:       base.VatRate,
		TaxLabel:      base.TaxLabel,
		FooterText:    base.FooterText,
		ShowQr:        base.ShowQr,
		QrSize:        base.QrSize,
		PaperSize:     base.PaperSize,
		RoundAmount:   base.RoundAmount,
	}

	storeInfo := receipthtml.StoreInfo{
		Name:        storeRecord.Name,
		Address:     storeRecord.Address,
		Phone:       storeRecord.Phone,
		TaxID:       storeRecord.TaxID,
		PromptPayID: storeRecord.PromptPayID,
		LogoURL:     storeRecord.LogoURL,
	}

	mockSale := buildMockSale(storeInfo, base)
	receiptBytes, err := receipthtml.RenderReceiptHTML(mockSale, storeInfo, cfg)
	if err != nil {
		return nil, err
	}
	return receipthtml.RenderPanelPreviewHTML(receiptBytes)
}

func buildMockSale(store receipthtml.StoreInfo, s ReceiptSettings) receipthtml.SaleData {
	subtotal := 67.0
	discountTotal := 3.35
	afterDiscount := subtotal - discountTotal
	vatAmount := 0.0
	grandTotal := afterDiscount
	if s.TaxMode == "exclusive" {
		vatAmount = math.Round(afterDiscount*(s.VatRate/100)*100) / 100
		grandTotal = afterDiscount + vatAmount
	}

	items := []receipthtml.SaleItem{
		{Name: "น้ำดื่มสิงห์ 600ml", SKU: "WTR001", Qty: 2, Price: 15.00, Total: 30.00},
		{Name: "โค้ก 325ml.", SKU: "SDR002", Qty: 1, Price: 20.00, Total: 20.00},
		{Name: "เลย์รสไนซ์ชี่ 45g.", SKU: "SNK003", Qty: 1, Price: 17.00, Total: 17.00},
	}

	now := time.Now()
	dateStr := fmt.Sprintf("%d/%d/%d %02d:%02d", now.Day(), int(now.Month()), now.Year()+543, now.Hour(), now.Minute())

	promptPayQR := ""
	if s.ShowQr && store.PromptPayID != "" {
		promptPayQR = receipthtml.PromptPayQRDataURI(store.PromptPayID, grandTotal)
	}

	displayGrandTotal := grandTotal
	if s.RoundAmount {
		displayGrandTotal = math.Round(grandTotal)
	}
	paid := math.Ceil(displayGrandTotal)

	return receipthtml.SaleData{
		OrderNo:        "INV-20250526",
		DateTime:       dateStr,
		CustomerName:   "ลูกค้าทั่วไป",
		Staff:          "Admin",
		Items:          items,
		TotalQty:       4,
		Subtotal:       subtotal,
		DiscountTotal:  discountTotal,
		AfterDiscount:  afterDiscount,
		VatAmount:      vatAmount,
		GrandTotal:     displayGrandTotal,
		GrandTotalText: "หกสิบสามบาทหกสิบห้าสตางค์",
		PaymentMethod:  "cash",
		PaymentLabel:   "เงินสด",
		Paid:           paid,
		Change:         math.Round((paid-displayGrandTotal)*100) / 100,
		PromptPayQRURI: template.URL(promptPayQR),
	}
}

func validHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func applyUpdate(s *ReceiptSettings, in UpdateReceiptSettingsRequest) {
	if in.DocumentTemplateKey != nil {
		s.DocumentTemplateKey = *in.DocumentTemplateKey
	}
	if in.DocumentDesign != nil {
		s.DocumentDesign = *in.DocumentDesign
	}
	if in.TemplateKey != nil {
		s.TemplateKey = *in.TemplateKey
	}
	if in.PaperSize != nil {
		s.PaperSize = *in.PaperSize
	}
	if in.PaperLength != nil {
		s.PaperLength = *in.PaperLength
	}
	if in.TaxMode != nil {
		s.TaxMode = *in.TaxMode
	}
	if in.VatRate != nil {
		s.VatRate = *in.VatRate
	}
	if in.TaxLabel != nil {
		s.TaxLabel = *in.TaxLabel
	}
	if in.ShowLogo != nil {
		s.ShowLogo = *in.ShowLogo
	}
	if in.LogoPosition != nil {
		s.LogoPosition = *in.LogoPosition
	}
	if in.ShowStoreName != nil {
		s.ShowStoreName = *in.ShowStoreName
	}
	if in.ShowAddress != nil {
		s.ShowAddress = *in.ShowAddress
	}
	if in.ShowPhone != nil {
		s.ShowPhone = *in.ShowPhone
	}
	if in.ShowTaxId != nil {
		s.ShowTaxId = *in.ShowTaxId
	}
	if in.FooterText != nil {
		s.FooterText = *in.FooterText
	}
	if len(in.PaymentChannels) > 0 {
		s.PaymentChannels = PaymentChannels(in.PaymentChannels)
	}
	if in.PrinterType != nil {
		s.PrinterType = *in.PrinterType
	}
	if in.PrinterName != nil {
		s.PrinterName = *in.PrinterName
	}
	if in.AutoPrint != nil {
		s.AutoPrint = *in.AutoPrint
	}
	if in.Copies != nil {
		s.Copies = *in.Copies
	}
	if in.ShowQr != nil {
		s.ShowQr = *in.ShowQr
	}
	if in.QrSize != nil {
		s.QrSize = *in.QrSize
	}
	if in.ShowCustomerDisplay != nil {
		s.ShowCustomerDisplay = *in.ShowCustomerDisplay
	}
	if in.ShowProductImages != nil {
		s.ShowProductImages = *in.ShowProductImages
	}
	if in.DateFormat != nil {
		s.DateFormat = *in.DateFormat
	}
	if in.TimeFormat != nil {
		s.TimeFormat = *in.TimeFormat
	}
	if in.CurrencyPosition != nil {
		s.CurrencyPosition = *in.CurrencyPosition
	}
	if in.RoundAmount != nil {
		s.RoundAmount = *in.RoundAmount
	}
}
