// Package receipthtml renders settings-aware thermal receipt HTML.
package receipthtml

import (
	"bytes"
	"fmt"
	"html/template"
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
	OrderNo          string
	DateTime         string
	CustomerName     string
	Staff            string
	Items            []SaleItem
	Subtotal         float64
	DiscountTotal    float64
	AfterDiscount    float64
	VatAmount        float64
	GrandTotal       float64
	GrandTotalText   string
	PaymentMethod    string
	PromptPayQRURI   template.URL // data:image/png;base64,... or "" — must be template.URL to avoid #ZgotmplZ sanitization
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
	case "80mm":
		return "80mm"
	case "a4":
		return "210mm"
	default:
		return "58mm"
	}
}

const receiptTpl = `<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>ใบเสร็จ {{.Sale.OrderNo}}</title>
<style>
  :root { color-scheme: light; }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    background: #ffffff;
    color: #1f2933;
    font-family: "Noto Sans Thai", "Tahoma", Arial, sans-serif;
    font-size: 11px;
    line-height: 1.35;
  }
  .receipt {
    width: 360px;
    margin: 0 auto;
    padding: 28px 24px 26px;
    background: #ffffff;
  }
  .center { text-align: center; }
  .right  { text-align: right; }
  .muted  { color: #606975; }
  .logo-wrap { margin-bottom: 10px; }
  .logo-wrap img { width: 60px; height: 60px; object-fit: contain; border-radius: 8px; }
  .logo-center { text-align: center; }
  .logo-left   { text-align: left; }
  .logo-right  { text-align: right; }
  .store-name  { color: #111827; font-size: 17px; font-weight: 700; line-height: 1.25; margin-bottom: 4px; }
  .store-meta  { font-size: 10.5px; line-height: 1.45; }
  .doc-title   { margin-top: 20px; color: #111827; font-size: 18px; font-weight: 700; line-height: 1.2; }
  .doc-subtitle{ margin-top: 2px; color: #6b7280; font-size: 11px; }
  .meta        { margin-top: 16px; display: grid; gap: 4px; }
  .meta-row    { display: grid; grid-template-columns: 72px 1fr; column-gap: 8px; }
  .meta-pair   { display: grid; grid-template-columns: 1fr 1fr; column-gap: 12px; }
  .meta-pair .meta-row:last-child { grid-template-columns: 54px 1fr; }
  .divider     { border-top: 1px solid #d9dde3; margin: 16px 0 10px; }
  table        { width: 100%; border-collapse: collapse; }
  th, td       { vertical-align: top; }
  th           { color: #606975; font-size: 10.5px; font-weight: 600; padding: 0 0 7px; border-bottom: 1px solid #d9dde3; }
  td           { padding: 6px 0; border-bottom: 1px solid #eef0f3; }
  .item-name   { width: 47%; }
  .qty         { width: 14%; text-align: center; }
  .money-col   { width: 19.5%; text-align: right; white-space: nowrap; }
  .summary     { margin-top: 10px; display: grid; gap: 6px; }
  .summary-row { display: grid; grid-template-columns: 1fr auto; gap: 12px; }
  .summary-total { margin-top: 2px; padding-top: 8px; border-top: 1px solid #d9dde3; color: #111827; font-size: 13px; font-weight: 700; }
  .footer-note { margin-top: 16px; padding: 8px 10px; border-radius: 8px; background: #f3f4f6; color: #374151; text-align: center; font-size: 11px; white-space: pre-line; }
  .qr-block    { margin-top: 16px; }
  .qr-block img { width: {{.QrWidth}}px; height: {{.QrWidth}}px; }
  @media print {
    @page { margin: 0; size: {{.PrintWidth}} auto; }
    body { background: #ffffff; }
    .receipt { width: {{.PrintWidth}}; padding: 7mm 6mm; }
  }
</style>
</head>
<body>
<main class="receipt">
  <header class="center">
    {{if and .Cfg.ShowLogo .Store.LogoURL}}
    <div class="logo-wrap logo-{{logoClass .Cfg.LogoPosition}}">
      <img src="{{.Store.LogoURL}}" alt="logo">
    </div>
    {{end}}
    {{if .Cfg.ShowStoreName}}<div class="store-name">{{.Store.Name}}</div>{{end}}
    <div class="store-meta muted">
      {{if .Cfg.ShowAddress}}{{.Store.Address}}<br>{{end}}
      {{if .Cfg.ShowTaxId}}เลขประจำตัวผู้เสียภาษี: {{or .Store.TaxID "-"}}<br>{{end}}
      {{if .Cfg.ShowPhone}}โทร: {{or .Store.Phone "-"}}{{end}}
    </div>
    <div class="doc-title">ใบเสร็จรับเงิน / ใบกำกับภาษีอย่างย่อ</div>
    <div class="doc-subtitle">(Receipt / Abbreviated Tax Invoice)</div>
  </header>

  <section class="meta">
    <div class="meta-row" style="grid-template-columns:auto 1fr;gap:6px">
      <span style="white-space:nowrap">เลขที่ (Doc No.):</span>
      <span class="right" style="word-break:break-all;font-size:9.5px;line-height:1.4">{{.Sale.OrderNo}}</span>
    </div>
    <div class="meta-pair">
      <div class="meta-row"><span>วันที่ (Date):</span><span></span></div>
      <div class="meta-row"><span></span><span class="right">{{.Sale.DateTime}}</span></div>
    </div>
    <div class="meta-pair">
      <div class="meta-row"><span>ลูกค้า (Customer):</span><span></span></div>
      <div class="meta-row"><span></span><span class="right">{{.Sale.CustomerName}}</span></div>
    </div>
  </section>

  <div class="divider"></div>

  <table aria-label="receipt items">
    <thead>
      <tr>
        <th class="item-name">รายการ (Item)</th>
        <th class="qty">จำนวน</th>
        <th class="money-col">ราคา</th>
        <th class="money-col">รวม</th>
      </tr>
    </thead>
    <tbody>
      {{range .Sale.Items}}
      <tr>
        <td class="item-name">{{.Name}}</td>
        <td class="qty">{{.Qty}}</td>
        <td class="money-col">{{money .Price}}</td>
        <td class="money-col">{{money .Total}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>

  <section class="summary">
    {{if gt .Sale.DiscountTotal 0.0}}
    <div class="summary-row muted"><span>ส่วนลด (Discount):</span><span>-{{money .Sale.DiscountTotal}} ฿</span></div>
    {{end}}
    {{if eq .Cfg.TaxMode "exclusive"}}
    <div class="summary-row"><span>มูลค่าก่อนภาษี:</span><span>{{money .Sale.AfterDiscount}} ฿</span></div>
    <div class="summary-row"><span>{{.Cfg.TaxLabel}} {{printf "%.0f" .Cfg.VatRate}}%:</span><span>{{money .Sale.VatAmount}} ฿</span></div>
    {{end}}
    {{if eq .Cfg.TaxMode "inclusive"}}
    <div class="summary-row muted"><span>ราคารวมภาษีแล้ว:</span><span>✓</span></div>
    {{end}}
    <div class="summary-row summary-total"><span>ยอดชำระสุทธิ (Net Total):</span><span>{{money .Sale.GrandTotal}} ฿</span></div>
  </section>

  <div class="footer-note">{{.Sale.GrandTotalText}}</div>

  {{if and .Cfg.ShowQr .Sale.PromptPayQRURI}}
  <div class="qr-block center">
    <b>สแกนเพื่อชำระเงิน (PromptPay)</b><br>
    <img src="{{.Sale.PromptPayQRURI}}" alt="PromptPay QR">
  </div>
  {{end}}

  {{if .Cfg.FooterText}}
  <div class="footer-note" style="margin-top:12px; background:transparent; border-top:1px dashed #d9dde3; padding-top:12px;">{{.Cfg.FooterText}}</div>
  {{end}}
</main>
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

	tpl, err := template.New("receipt").Funcs(template.FuncMap{
		"money": func(v float64) string { return fmt.Sprintf("%.2f", v) },
		"logoClass": logoClassFn,
		"or":        orFn,
		"printf":    fmt.Sprintf,
	}).Parse(receiptTpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"Sale":       sale,
		"Store":      store,
		"Cfg":        cfg,
		"QrWidth":    qrWidth(cfg.QrSize),
		"PrintWidth": printWidth(cfg.PaperSize),
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
