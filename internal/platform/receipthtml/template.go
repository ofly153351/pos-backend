// Package receipthtml renders settings-aware thermal receipt HTML.
package receipthtml

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"strings"
)

// Config holds receipt_settings fields that control rendering.
type Config struct {
	ShowLogo      bool
	LogoPosition  string // "top_center" | "top_left" | "top_right"
	LogoURL       string
	ShowStoreName bool
	ShowAddress   bool
	ShowPhone     bool
	ShowTaxId     bool
	TaxMode       string  // "none" | "inclusive" | "exclusive"
	VatRate       float64 // e.g. 7.0
	TaxLabel      string
	FooterText    string
	ShowQr        bool
	QrSize        string // "small" | "medium" | "large"
	PaperSize     string // "58mm" | "80mm" | "a4"
	RoundAmount   bool   // when true, display monetary totals as whole baht (no satang)
}

// StoreInfo carries the store record fields used in the receipt header.
type StoreInfo struct {
	Name        string
	Address     string
	Phone       string
	TaxID       string
	PromptPayID string
	LogoURL     string
}

// SaleItem is one line on the receipt.
type SaleItem struct {
	Name  string
	SKU   string
	Qty   int
	Price float64
	Total float64
}

// SaleData is the sale record used for rendering.
type SaleData struct {
	OrderNo        string
	DateTime       string
	CustomerName   string
	Staff          string
	Items          []SaleItem
	TotalQty       int // sum of all item quantities (ชิ้น)
	Subtotal       float64
	DiscountTotal  float64
	AfterDiscount  float64
	VatAmount      float64
	GrandTotal     float64
	GrandTotalText string
	PaymentMethod  string
	PaymentLabel   string  // Thai display label (เงินสด / พร้อมเพย์)
	Paid           float64 // amount received
	Change         float64 // change given
	PromptPayQRURI template.URL // data:image/png;base64,... or "" — must be template.URL to avoid #ZgotmplZ sanitization
}

func qrWidth(size string) int {
	switch size {
	case "small":
		return 100
	case "large":
		return 200
	default:
		return 150
	}
}

func printWidth(paperSize string) string {
	switch paperSize {
	case "a4":
		return "210mm"
	default: // "80mm" and anything else
		return "80mm"
	}
}

// paperWidth returns the on-screen/render width of the receipt body.
func paperWidth(paperSize string) string {
	switch paperSize {
	case "a4":
		return "190mm"
	default: // 80mm thermal
		return "80mm"
	}
}

// formatThousands formats a float as "1,234.56" with comma thousands separators.
func formatThousands(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	intPart, decPart := s, ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart, decPart = s[:dot], s[dot:]
	}
	neg := strings.HasPrefix(intPart, "-")
	if neg {
		intPart = intPart[1:]
	}
	var b strings.Builder
	n := len(intPart)
	for i, c := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := b.String() + decPart
	if neg {
		out = "-" + out
	}
	return out
}

// formatWholeBaht formats a float as "1,234" (no decimal) with comma thousands separators.
func formatWholeBaht(v float64) string {
	s := fmt.Sprintf("%.0f", math.Round(v))
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	n := len(s)
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := b.String()
	if neg {
		out = "-" + out
	}
	return out
}

// PaymentLabel maps a payment-method code to a Thai display label.
// Canonical — mirrors lib/payment-method.ts on the frontend so the printed receipt
// matches the dashboard / reports. Legacy aliases collapse onto the canonical channel;
// `credit` (sold-on-credit) is its own label, NOT a card.
func PaymentLabel(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "cash":
		return "เงินสด"
	case "bank_transfer", "transfer":
		return "โอนเงิน"
	case "promptpay", "qr":
		return "พร้อมเพย์ (QR)"
	case "card", "credit_card", "debit_card", "debit":
		return "บัตรเครดิต / เดบิต"
	case "credit":
		return "ขายเชื่อ / ค้างชำระ"
	case "truemoney":
		return "TrueMoney Wallet"
	case "shopeepay":
		return "ShopeePay"
	case "":
		return "-"
	default:
		return method
	}
}

const receiptTpl = `<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>ใบเสร็จรับเงิน — {{.Sale.OrderNo}}</title>
<style>
  *{box-sizing:border-box;margin:0;padding:0}

  body{
    font-family:'Sarabun','Noto Sans Thai','Tahoma',sans-serif;
    color:#000;
    -webkit-font-smoothing:antialiased;
  }

  @media screen{
    body{background:#fff}
  }

  @media print{
    body{background:#fff}
    @page{size:{{.PrintWidth}} auto;margin:0}
  }

  .receipt{
    width:{{.PaperWidth}};
    margin:0 auto;
    padding:5mm 4mm 6mm;
    font-size:12.5px;
    line-height:1.45;
    background:#fff;
  }

  /* Header */
  .center{text-align:center}
  .logo-wrap{margin-bottom:6px}
  .logo-wrap img{max-height:48px;max-width:140px;object-fit:contain}
  .logo-left{text-align:left}
  .logo-right{text-align:right}
  .store-name{font-size:20px;font-weight:800;letter-spacing:.3px;margin-bottom:2px}
  .store-meta{font-size:12px;line-height:1.5}

  .divider{border:none;border-top:1.5px dashed #000;margin:7px 0}

  .doc-title{font-size:16px;font-weight:800;margin-bottom:1px}
  .doc-title-en{font-size:12px;font-weight:400;color:#222}

  /* Info rows */
  .info{font-size:12.5px}
  .info-row{display:flex;margin-bottom:2px}
  .info-label{width:64px;flex-shrink:0;font-weight:600}
  .info-value{flex:1;word-break:break-word}
  .mono{font-variant-numeric:tabular-nums}

  /* Items */
  .item{padding:5px 0;border-bottom:1px dashed #aaa}
  .item:first-child{padding-top:2px}
  .item-line1{display:flex;justify-content:space-between;align-items:baseline;gap:8px}
  .item-name{font-weight:600}
  .item-amount{font-weight:600;white-space:nowrap;font-variant-numeric:tabular-nums}
  .item-qty{padding-left:16px;color:#222;font-size:12px;margin-top:1px;font-variant-numeric:tabular-nums}

  /* Count */
  .count-row{display:flex;justify-content:space-between;padding:5px 0 0;font-size:12.5px}
  .count-pieces{font-variant-numeric:tabular-nums}

  /* Totals */
  .total-row{display:flex;justify-content:space-between;margin-bottom:2px;font-size:12.5px}
  .total-row .v{font-variant-numeric:tabular-nums}

  .grand{display:flex;justify-content:space-between;align-items:baseline;margin:4px 0}
  .grand .l{font-size:18px;font-weight:800}
  .grand .v{font-size:20px;font-weight:800;font-variant-numeric:tabular-nums}

  /* QR */
  .qr-block{text-align:center;margin:8px 0}
  .qr-block img{width:{{.QrWidth}}px;height:{{.QrWidth}}px}

  /* Footer */
  .footer{text-align:center;font-size:13px;line-height:1.7;white-space:pre-line}
  .footer .thanks{font-weight:600}
</style>
</head>
<body>
<div class="receipt">

  <!-- HEADER -->
  <div class="center">
    {{if and .Cfg.ShowLogo .Store.LogoURL}}
    <div class="logo-wrap logo-{{logoClass .Cfg.LogoPosition}}"><img src="{{.Store.LogoURL}}" alt="logo"></div>
    {{end}}
    {{if .Cfg.ShowStoreName}}<div class="store-name">{{.Store.Name}}</div>{{end}}
    {{if .Cfg.ShowAddress}}<div class="store-meta">{{.Store.Address}}</div>{{end}}
    {{if .Cfg.ShowPhone}}<div class="store-meta">โทร: {{or .Store.Phone "-"}}</div>{{end}}
    {{if .Cfg.ShowTaxId}}<div class="store-meta">เลขประจำตัวผู้เสียภาษี: {{or .Store.TaxID "-"}}</div>{{end}}
  </div>

  <hr class="divider"/>

  <div class="center">
    <div class="doc-title">ใบเสร็จรับเงิน / ใบกำกับภาษีอย่างย่อ</div>
    <div class="doc-title-en">Receipt / Abbreviated Tax Invoice</div>
  </div>

  <hr class="divider"/>

  <!-- INFO -->
  <div class="info">
    <div class="info-row"><span class="info-label">เลขที่:</span><span class="info-value mono">{{.Sale.OrderNo}}</span></div>
    <div class="info-row"><span class="info-label">วันที่:</span><span class="info-value mono">{{.Sale.DateTime}}</span></div>
    <div class="info-row"><span class="info-label">ลูกค้า:</span><span class="info-value">{{.Sale.CustomerName}}</span></div>
    <div class="info-row"><span class="info-label">พนักงาน:</span><span class="info-value">{{.Sale.Staff}}</span></div>
    <div class="info-row"><span class="info-label">ชำระโดย:</span><span class="info-value">{{.Sale.PaymentLabel}}</span></div>
  </div>

  <hr class="divider"/>

  <!-- ITEMS -->
  <div class="items">
    {{range $i, $it := .Sale.Items}}
    <div class="item">
      <div class="item-line1"><span class="item-name">{{add $i 1}}. {{$it.Name}}</span><span class="item-amount">{{baht $it.Total}}</span></div>
      <div class="item-qty">{{$it.Qty}} x {{money $it.Price}}</div>
    </div>
    {{end}}
  </div>

  <hr class="divider"/>

  <!-- COUNT -->
  <div class="count-row">
    <span>รายการทั้งหมด {{len .Sale.Items}} รายการ</span>
    <span class="count-pieces">{{.Sale.TotalQty}} ชิ้น</span>
  </div>

  <hr class="divider"/>

  <!-- TOTALS -->
  <div class="totals">
    {{if gt .Sale.DiscountTotal 0.0}}
    <div class="total-row"><span>ยอดรวมสินค้า</span><span class="v">{{baht .Sale.Subtotal}}</span></div>
    <div class="total-row"><span>ส่วนลด</span><span class="v">-{{baht .Sale.DiscountTotal}}</span></div>
    {{end}}
    {{if gt .Sale.VatAmount 0.0}}
    <div class="total-row"><span>รวมก่อน VAT</span><span class="v">{{baht .Sale.AfterDiscount}}</span></div>
    <div class="total-row"><span>{{.Cfg.TaxLabel}} {{printf "%.0f" .Cfg.VatRate}}%</span><span class="v">{{baht .Sale.VatAmount}}</span></div>
    {{end}}
  </div>

  <hr class="divider"/>

  <div class="grand">
    <span class="l">ยอดสุทธิ</span>
    <span class="v">{{bahtTotal .Sale.GrandTotal}}</span>
  </div>
  {{if gt .Sale.Paid 0.0}}
  <div class="total-row"><span>รับเงิน</span><span class="v">{{bahtTotal .Sale.Paid}}</span></div>
  <div class="total-row"><span>เงินทอน</span><span class="v">{{bahtTotal .Sale.Change}}</span></div>
  {{end}}

  <hr class="divider"/>

  {{if and .Cfg.ShowQr .Sale.PromptPayQRURI}}
  <div class="qr-block">
    <div style="font-weight:600;margin-bottom:4px">สแกนเพื่อชำระเงิน (PromptPay)</div>
    <img src="{{.Sale.PromptPayQRURI}}" alt="PromptPay QR">
  </div>
  <hr class="divider"/>
  {{end}}

  <!-- FOOTER -->
  <div class="footer">
    {{if .Cfg.FooterText}}<div>{{.Cfg.FooterText}}</div>{{else}}<div>ขอบคุณที่ใช้บริการ</div>{{end}}
    <div class="thanks">*** ขอบคุณครับ ***</div>
  </div>

</div>
</body>
</html>`

const previewWrapperTpl = `<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<title>Receipt Preview</title>
<style>
  * { box-sizing: border-box; }
  body { margin: 0; min-height: 100vh; background: rgba(17,24,39,0.42); font-family: Arial,sans-serif; }
  .toolbar { width: 384px; margin: 0 auto; padding: 0 24px 24px; display: flex; gap: 8px; align-items: center; justify-content: center; }
  .btn { min-width: 132px; height: 42px; border-radius: 10px; border: 0; background: #fffdf4; color: #374151; padding: 0 18px; font-size: 14px; font-weight: 700; }
  .btn-primary { background: #7c3aed; color: #fff; box-shadow: 0 8px 18px rgba(124,58,237,0.34); }
  .stage { padding: 28px 12px 0; }
  .paper { width: 384px; margin: 0 auto; background: #fff; border-radius: 22px; box-shadow: 0 18px 42px rgba(15,23,42,0.18); overflow: hidden; }
  @media print {
    .toolbar { display: none; }
    .stage { padding: 0; background: #fff; }
    .paper { box-shadow: none; border: 0; width: auto; }
    body { background: #fff; }
  }
</style>
</head>
<body>
  <div class="stage"><div class="paper">{{.ReceiptHTML}}</div></div>
  <div class="toolbar">
    <button class="btn btn-primary" onclick="window.print()">พิมพ์ใบเสร็จ</button>
    <button class="btn" onclick="window.close()">ปิดหน้าต่าง</button>
  </div>
</body>
</html>`

// RenderReceiptHTML renders the receipt inner HTML (no wrapper).
func RenderReceiptHTML(sale SaleData, store StoreInfo, cfg Config) ([]byte, error) {
	logoClassFn := func(pos string) string {
		switch pos {
		case "top_left":
			return "left"
		case "top_right":
			return "right"
		default:
			return "center"
		}
	}
	orFn := func(a, b string) string {
		if strings.TrimSpace(a) == "" {
			return b
		}
		return a
	}

	bahtTotalFn := func(v float64) string { return "฿" + formatThousands(v) }
	if cfg.RoundAmount {
		bahtTotalFn = func(v float64) string { return "฿" + formatWholeBaht(v) }
	}
	tpl, err := template.New("receipt").Funcs(template.FuncMap{
		"money":     func(v float64) string { return fmt.Sprintf("%.2f", v) },
		"baht":      func(v float64) string { return "฿" + formatThousands(v) },
		"bahtTotal": bahtTotalFn,
		"logoClass": logoClassFn,
		"or":        orFn,
		"printf":    fmt.Sprintf,
		"add":       func(a, b int) int { return a + b },
		"len":       func(items []SaleItem) int { return len(items) },
	}).Parse(receiptTpl)
	if err != nil {
		return nil, err
	}

	// Auto-fill PaymentLabel + TotalQty when caller left them blank.
	if sale.PaymentLabel == "" {
		sale.PaymentLabel = PaymentLabel(sale.PaymentMethod)
	}
	if sale.TotalQty == 0 {
		for _, it := range sale.Items {
			sale.TotalQty += it.Qty
		}
	}

	data := map[string]any{
		"Sale":       sale,
		"Store":      store,
		"Cfg":        cfg,
		"QrWidth":    qrWidth(cfg.QrSize),
		"PrintWidth": printWidth(cfg.PaperSize),
		"PaperWidth": paperWidth(cfg.PaperSize),
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const panelWrapperTpl = `<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  html, body { background: #ffffff; }
  ::-webkit-scrollbar { width: 3px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb { background: #d1d5db; border-radius: 99px; }
  ::-webkit-scrollbar-thumb:hover { background: #9ca3af; }
  * { scrollbar-width: thin; scrollbar-color: #d1d5db transparent; }
</style>
</head>
<body>{{.ReceiptHTML}}</body>
</html>`

// RenderPanelPreviewHTML wraps receipt HTML in a minimal shell for embedded panel iframes.
// No dark background, no paper box-shadow, no toolbar buttons.
func RenderPanelPreviewHTML(receiptHTML []byte) ([]byte, error) {
	tpl, err := template.New("panel").Parse(panelWrapperTpl)
	if err != nil {
		return nil, err
	}
	data := struct{ ReceiptHTML template.HTML }{
		ReceiptHTML: template.HTML(string(receiptHTML)),
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderPreviewHTML wraps receipt HTML in the preview shell.
func RenderPreviewHTML(receiptHTML []byte) ([]byte, error) {
	tpl, err := template.New("preview").Parse(previewWrapperTpl)
	if err != nil {
		return nil, err
	}
	data := struct{ ReceiptHTML template.HTML }{
		ReceiptHTML: template.HTML(string(receiptHTML)),
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
