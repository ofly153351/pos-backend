package sale

import (
	"context"
	"fmt"
	"html/template"
	"math"
	"os"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/receipthtml"
)

// SettingsRepository is the minimal interface the sale service needs from receipt_settings.
type SettingsRepository interface {
	GetByStoreID(ctx context.Context, storeID string) (ReceiptSettingsView, error)
}

// ReceiptSettingsView is the subset of receipt_settings the sale service cares about.
type ReceiptSettingsView struct {
	ShowLogo      bool
	LogoPosition  string
	ShowStoreName bool
	ShowAddress   bool
	ShowPhone     bool
	ShowTaxId     bool
	TaxMode       string
	VatRate       float64
	TaxLabel      string
	FooterText    string
	ShowQr        bool
	QrSize        string
	PaperSize     string
	RoundAmount   bool
}

func defaultSettings() ReceiptSettingsView {
	return ReceiptSettingsView{
		ShowLogo: false, LogoPosition: "top_center",
		ShowStoreName: true, ShowAddress: true, ShowPhone: true, ShowTaxId: true,
		TaxMode: "exclusive", VatRate: 7, TaxLabel: "ภาษีมูลค่าเพิ่ม (VAT 7%)",
		FooterText: "ขอบคุณที่ใช้บริการ",
		ShowQr: true, QrSize: "medium", PaperSize: "80mm",
		RoundAmount: true,
	}
}

func (s Service) loadSettings(ctx context.Context, storeID string) ReceiptSettingsView {
	if s.settingsRepo == nil {
		return defaultSettings()
	}
	sv, err := s.settingsRepo.GetByStoreID(ctx, storeID)
	if err != nil {
		return defaultSettings()
	}
	return sv
}

func (s Service) GenerateReceiptHTML(ctx context.Context, actor auth.Claims, storeID, saleID string) ([]byte, error) {
	saleRecord, err := s.GetByID(ctx, actor, storeID, saleID)
	if err != nil {
		return nil, err
	}
	sv := s.loadSettings(ctx, storeID)
	return renderSaleAsReceiptHTML(saleRecord, sv)
}

func (s Service) GenerateReceiptPreviewHTML(ctx context.Context, actor auth.Claims, storeID, saleID string) ([]byte, error) {
	receiptHTML, err := s.GenerateReceiptHTML(ctx, actor, storeID, saleID)
	if err != nil {
		return nil, err
	}
	return receipthtml.RenderPreviewHTML(receiptHTML)
}

// renderSaleAsReceiptHTML maps Sale + settings → receipt HTML bytes.
func renderSaleAsReceiptHTML(s Sale, sv ReceiptSettingsView) ([]byte, error) {
	vatPercent := s.VATPercent
	if vatPercent < 0 {
		vatPercent = 0
	}

	// Use authoritative stored totals (computed by repository at sale time) so the
	// receipt always matches the amount the customer was actually charged.
	grandTotal := roundMoney(s.TotalAmount)
	vatAmount := roundMoney(s.VATAmount)
	if sv.TaxMode != "exclusive" {
		vatAmount = 0
	}
	// afterDiscount keeps full 2dp precision for transparency on the receipt.
	afterDiscount := roundMoney(grandTotal - vatAmount)
	if afterDiscount < 0 {
		afterDiscount = 0
	}
	discountTotal := roundMoney(s.DiscountAmount)
	if discountTotal < 0 {
		discountTotal = 0
	}

	// Display rounding: only the grand total / paid / change lines are rounded.
	// Subtotal, VAT, and item lines keep 2dp precision.
	displayGrandTotal := grandTotal
	displayChange := roundMoney(s.ChangeAmount)
	if sv.RoundAmount {
		displayGrandTotal = math.Round(grandTotal)
		raw := s.PaidAmount - displayGrandTotal
		if raw >= 0 {
			displayChange = roundMoney(raw)
		} else {
			displayChange = 0
		}
	}

	items := make([]receipthtml.SaleItem, 0, len(s.Items))
	for _, it := range s.Items {
		items = append(items, receipthtml.SaleItem{
			Name:  strings.TrimSpace(it.ProductName),
			SKU:   strings.TrimSpace(it.SKU),
			Qty:   it.Quantity,
			Price: roundMoney(it.UnitPrice),
			Total: roundMoney(it.LineTotal),
		})
	}

	soldAt := s.SoldAt.In(time.Local)
	if soldAt.IsZero() {
		soldAt = time.Now()
	}

	customerName := strings.TrimSpace(s.CustomerName)
	if customerName == "" {
		customerName = "ลูกค้าทั่วไป (เงินสด)"
	}
	staff := strings.TrimSpace(s.CashierName)
	if staff == "" {
		staff = s.CashierUserID
	}

	promptPayID := strings.TrimSpace(s.StorePromptPayID)
	promptPayQR := ""
	// Only generate QR when: setting enabled + ID present + generated URI is non-empty
	if sv.ShowQr && promptPayID != "" {
		uri := receipthtml.PromptPayQRDataURI(promptPayID, grandTotal)
		if uri != "" {
			promptPayQR = uri
		}
	}

	totalQty := 0
	for _, it := range items {
		totalQty += it.Qty
	}

	sale := receipthtml.SaleData{
		OrderNo:        fallback(s.SaleNumber, s.ID),
		DateTime:       formatThaiDateTime(soldAt),
		CustomerName:   customerName,
		Staff:          fallback(staff, "-"),
		Items:          items,
		TotalQty:       totalQty,
		Subtotal:       roundMoney(s.SubtotalAmount),
		DiscountTotal:  discountTotal,
		AfterDiscount:  afterDiscount,
		VatAmount:      vatAmount,
		GrandTotal:     displayGrandTotal,
		GrandTotalText: formatThaiBahtText(displayGrandTotal),
		PaymentMethod:  fallback(s.PaymentMethod, "-"),
		PaymentLabel:   receipthtml.PaymentLabel(s.PaymentMethod),
		Paid:           roundMoney(s.PaidAmount),
		Change:         displayChange,
		PromptPayQRURI: template.URL(promptPayQR),
	}

	store := receipthtml.StoreInfo{
		Name:        fallback(s.StoreName, "-"),
		Address:     fallback(s.StoreAddress, "-"),
		Phone:       fallback(s.StorePhone, "-"),
		TaxID:       fallback(s.StoreTaxID, fallback(os.Getenv("APP_STORE_TAX_ID"), "-")),
		PromptPayID: promptPayID,
		LogoURL:     strings.TrimSpace(s.StoreLogoURL),
	}

	cfg := receipthtml.Config{
		ShowLogo:      sv.ShowLogo,
		LogoPosition:  sv.LogoPosition,
		LogoURL:       store.LogoURL,
		ShowStoreName: sv.ShowStoreName,
		ShowAddress:   sv.ShowAddress,
		ShowPhone:     sv.ShowPhone,
		ShowTaxId:     sv.ShowTaxId,
		TaxMode:       sv.TaxMode,
		VatRate:       vatPercent,
		TaxLabel:      sv.TaxLabel,
		FooterText:    sv.FooterText,
		ShowQr:        sv.ShowQr,
		QrSize:        sv.QrSize,
		PaperSize:     sv.PaperSize,
		RoundAmount:   sv.RoundAmount,
	}

	return receipthtml.RenderReceiptHTML(sale, store, cfg)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func formatThaiDate(t time.Time) string {
	return fmt.Sprintf("%d/%d/%d", t.Day(), int(t.Month()), t.Year()+543)
}

func formatThaiDateTime(t time.Time) string {
	return fmt.Sprintf("%d/%d/%d %02d:%02d", t.Day(), int(t.Month()), t.Year()+543, t.Hour(), t.Minute())
}

func formatThaiBahtText(v float64) string {
	totalSatang := int64(math.Round(v * 100))
	if totalSatang < 0 {
		totalSatang = 0
	}
	baht := totalSatang / 100
	satang := totalSatang % 100
	text := thaiNumberText(baht) + "บาท"
	if satang == 0 {
		return text + "ถ้วน"
	}
	return text + thaiNumberText(satang) + "สตางค์"
}

func thaiNumberText(n int64) string {
	if n == 0 {
		return "ศูนย์"
	}
	if n >= 1000000 {
		m := n / 1000000
		r := n % 1000000
		t := thaiNumberText(m) + "ล้าน"
		if r > 0 {
			t += thaiNumberText(r)
		}
		return t
	}
	digits := []string{"", "หนึ่ง", "สอง", "สาม", "สี่", "ห้า", "หก", "เจ็ด", "แปด", "เก้า"}
	positions := []string{"", "สิบ", "ร้อย", "พัน", "หมื่น", "แสน"}
	raw := fmt.Sprintf("%d", n)
	var out strings.Builder
	for i, r := range raw {
		digit := int(r - '0')
		if digit == 0 {
			continue
		}
		pos := len(raw) - i - 1
		switch pos {
		case 0:
			if digit == 1 && len(raw) > 1 {
				out.WriteString("เอ็ด")
			} else {
				out.WriteString(digits[digit])
			}
		case 1:
			switch digit {
			case 1:
				out.WriteString("สิบ")
			case 2:
				out.WriteString("ยี่สิบ")
			default:
				out.WriteString(digits[digit])
				out.WriteString("สิบ")
			}
		default:
			out.WriteString(digits[digit])
			out.WriteString(positions[pos])
		}
	}
	return out.String()
}

func fallback(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

