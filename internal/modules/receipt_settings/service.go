package receipt_settings

import (
	"context"
	"fmt"
	"html/template"
	"math"
	"time"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/store"
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
	ok, err := s.storeRepo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ReceiptSettings{}, err
	}
	if !ok {
		return ReceiptSettings{}, ErrForbidden
	}

	settings, err := s.repo.GetByStoreID(ctx, storeID)
	if err == ErrNotFound {
		return s.createDefaults(ctx, storeID)
	}
	return settings, err
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID string, input UpdateReceiptSettingsRequest) (ReceiptSettings, error) {
	ok, err := s.storeRepo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ReceiptSettings{}, err
	}
	if !ok {
		return ReceiptSettings{}, ErrForbidden
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

	applyUpdate(&current, input)
	return s.repo.Upsert(ctx, current)
}

func (s Service) createDefaults(ctx context.Context, storeID string) (ReceiptSettings, error) {
	now := time.Now().UTC()
	defaults := ReceiptSettings{
		ID:                  newID(),
		StoreID:             storeID,
		TemplateKey:         "modern_classic",
		PaperSize:           "58mm",
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

func defaultChannels() PaymentChannels {
	return PaymentChannels{
		{Key: "cash", Enabled: true},
		{Key: "card", Enabled: true},
		{Key: "qr", Enabled: true},
		{Key: "promptpay", Enabled: true},
		{Key: "truemoney", Enabled: false},
		{Key: "shopeepay", Enabled: false},
	}
}

// PreviewHTML renders a preview receipt using the provided settings override + real store info.
func (s Service) PreviewHTML(ctx context.Context, actor auth.Claims, storeID string, input UpdateReceiptSettingsRequest) ([]byte, error) {
	ok, err := s.storeRepo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}

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
	}

	storeInfo := receipthtml.StoreInfo{
		Name:        storeRecord.Name,
		Address:     storeRecord.Address,
		Phone:       storeRecord.Phone,
		TaxID:       "",
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
		// For preview we skip actual QR generation — show placeholder text
		promptPayQR = ""
	}

	return receipthtml.SaleData{
		OrderNo:        "INV-20250526",
		DateTime:       dateStr,
		CustomerName:   "ลูกค้าทั่วไป (เงินสด)",
		Staff:          "Admin",
		Items:          items,
		Subtotal:       subtotal,
		DiscountTotal:  discountTotal,
		AfterDiscount:  afterDiscount,
		VatAmount:      vatAmount,
		GrandTotal:     grandTotal,
		GrandTotalText: "หกสิบสามบาทหกสิบห้าสตางค์",
		PaymentMethod:  "เงินสด",
		PromptPayQRURI: template.URL(promptPayQR),
	}
}

func applyUpdate(s *ReceiptSettings, in UpdateReceiptSettingsRequest) {
	if in.TemplateKey != nil         { s.TemplateKey = *in.TemplateKey }
	if in.PaperSize != nil           { s.PaperSize = *in.PaperSize }
	if in.PaperLength != nil         { s.PaperLength = *in.PaperLength }
	if in.TaxMode != nil             { s.TaxMode = *in.TaxMode }
	if in.VatRate != nil             { s.VatRate = *in.VatRate }
	if in.TaxLabel != nil            { s.TaxLabel = *in.TaxLabel }
	if in.ShowLogo != nil            { s.ShowLogo = *in.ShowLogo }
	if in.LogoPosition != nil        { s.LogoPosition = *in.LogoPosition }
	if in.ShowStoreName != nil       { s.ShowStoreName = *in.ShowStoreName }
	if in.ShowAddress != nil         { s.ShowAddress = *in.ShowAddress }
	if in.ShowPhone != nil           { s.ShowPhone = *in.ShowPhone }
	if in.ShowTaxId != nil           { s.ShowTaxId = *in.ShowTaxId }
	if in.FooterText != nil          { s.FooterText = *in.FooterText }
	if len(in.PaymentChannels) > 0   { s.PaymentChannels = PaymentChannels(in.PaymentChannels) }
	if in.PrinterType != nil         { s.PrinterType = *in.PrinterType }
	if in.PrinterName != nil         { s.PrinterName = *in.PrinterName }
	if in.AutoPrint != nil           { s.AutoPrint = *in.AutoPrint }
	if in.Copies != nil              { s.Copies = *in.Copies }
	if in.ShowQr != nil              { s.ShowQr = *in.ShowQr }
	if in.QrSize != nil              { s.QrSize = *in.QrSize }
	if in.ShowCustomerDisplay != nil { s.ShowCustomerDisplay = *in.ShowCustomerDisplay }
	if in.ShowProductImages != nil   { s.ShowProductImages = *in.ShowProductImages }
	if in.DateFormat != nil          { s.DateFormat = *in.DateFormat }
	if in.TimeFormat != nil          { s.TimeFormat = *in.TimeFormat }
	if in.CurrencyPosition != nil    { s.CurrencyPosition = *in.CurrencyPosition }
}
