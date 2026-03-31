package sale

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"pos-backend/internal/modules/auth"
)

const receiptTemplate = `<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Receipt {{index . "order" "order_no"}}</title>
<style>
  body {
    font-family: "Courier New", "Noto Sans Thai", "Tahoma", monospace;
    width: 300px;
    margin: auto;
    color: #111;
    line-height: 1.35;
    font-size: 13px;
  }
  .center { text-align: center; }
  .right { text-align: right; }
  .line { border-top: 1px dashed #000; margin: 8px 0; }
  table { width: 100%; border-collapse: collapse; }
  td { vertical-align: top; }
</style>
</head>
<body>
<div class="center">
  <b>{{index . "store" "name"}}</b><br>
  {{index . "store" "address"}}<br>
  Tax# {{index . "store" "tax_id"}} ({{if index . "store" "vat_included"}}Vat Included{{else}}Vat Excluded{{end}})
</div>

<div class="line"></div>

Order#: {{index . "order" "order_no"}}<br>
Staff: {{index . "order" "staff"}}<br>
Date: {{index . "order" "datetime"}}<br>

<div class="line"></div>

<div><b>ใบเสร็จรับเงิน / ใบกำกับภาษีอย่างย่อ</b></div>

<div class="line"></div>

{{range $item := index . "items"}}
<table>
<tr>
  <td>{{index $item "code"}}</td>
  <td class="right">{{money (index $item "total")}}</td>
</tr>
<tr>
  <td colspan="2">{{index $item "name"}}</td>
</tr>
<tr>
  <td>x {{index $item "qty"}}</td>
  <td class="right">{{money (index $item "price")}}</td>
</tr>
</table>
{{end}}

<div class="line"></div>

<table>
<tr><td>รวม</td><td class="right">{{money (index . "summary" "subtotal")}}</td></tr>
<tr><td>ส่วนลด</td><td class="right">{{money (index . "summary" "discount_bill")}}</td></tr>
<tr><td>VAT {{index . "summary" "vat_percent"}}%</td><td class="right">{{money (index . "summary" "vat_amount")}}</td></tr>
<tr><td><b>รวมสุทธิ</b></td><td class="right"><b>{{money (index . "summary" "grand_total")}}</b></td></tr>
</table>

<div class="line"></div>

รับเงิน: {{money (index . "payment" "received")}}<br>
เงินทอน: {{money (index . "payment" "change")}}<br>
ชำระโดย: {{index . "payment" "method"}}<br>
{{if index . "payment" "promptpay_id"}}พร้อมเพย์: {{index . "payment" "promptpay_id"}}<br>{{end}}

{{if index . "payment" "promptpay_qr_data_uri"}}
<div class="center">
  <br><b>สแกนเพื่อชำระเงิน</b><br>
  <img src="{{index . "payment" "promptpay_qr_data_uri"}}" alt="PromptPay QR" width="180" height="180"><br>
</div>
{{end}}

<div class="center">
  <br>{{index . "footer" "message"}}<br><br>
</div>

Ref#: {{index . "footer" "ref_no"}}<br>
Print: {{index . "footer" "print_date"}}<br>
Tel: {{index . "footer" "contact"}}
</body>
</html>`

const receiptPreviewTemplate = `<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Receipt Preview</title>
<style>
  :root { color-scheme: light; }
  body {
    margin: 0;
    background: #f3f4f6;
    color: #111827;
    font-family: Arial, Helvetica, sans-serif;
  }
  .toolbar {
    position: sticky;
    top: 0;
    z-index: 10;
    background: #ffffff;
    border-bottom: 1px solid #e5e7eb;
    padding: 10px 12px;
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
  }
  .title {
    font-size: 14px;
    font-weight: 600;
  }
  .actions { display: flex; gap: 8px; }
  .btn {
    height: 34px;
    border-radius: 8px;
    border: 1px solid #d1d5db;
    background: #ffffff;
    padding: 0 12px;
    cursor: pointer;
    font-size: 13px;
  }
  .btn-primary {
    border-color: #111827;
    background: #111827;
    color: #ffffff;
  }
  .stage { padding: 20px 12px 28px; }
  .paper {
    width: 320px;
    margin: 0 auto;
    background: #ffffff;
    border: 1px solid #d1d5db;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.08);
    padding: 8px;
  }
  @media print {
    .toolbar { display: none; }
    .stage {
      padding: 0;
      background: #ffffff;
    }
    .paper {
      box-shadow: none;
      border: 0;
      padding: 0;
      width: auto;
    }
    body { background: #ffffff; }
  }
</style>
</head>
<body>
  <div class="toolbar">
    <div class="title">Receipt Preview</div>
    <div class="actions">
      <button class="btn" type="button" onclick="window.close()">Close</button>
      <button class="btn btn-primary" type="button" onclick="window.print()">Print</button>
    </div>
  </div>
  <div class="stage">
    <div class="paper">{{.ReceiptHTML}}</div>
  </div>
</body>
</html>`

func (s Service) GenerateReceiptHTML(ctx context.Context, actor auth.Claims, storeID, saleID string) ([]byte, error) {
	saleRecord, err := s.GetByID(ctx, actor, storeID, saleID)
	if err != nil {
		return nil, err
	}

	payload := buildReceiptPayload(saleRecord)
	tpl, err := template.New("receipt").Funcs(template.FuncMap{
		"money": func(v any) string {
			switch n := v.(type) {
			case float64:
				return formatMoney(n)
			case float32:
				return formatMoney(float64(n))
			default:
				return "0.00"
			}
		},
	}).Parse(receiptTemplate)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, payload); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (s Service) GenerateReceiptPreviewHTML(ctx context.Context, actor auth.Claims, storeID, saleID string) ([]byte, error) {
	receipt, err := s.GenerateReceiptHTML(ctx, actor, storeID, saleID)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("receipt_preview").Parse(receiptPreviewTemplate)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	view := struct {
		ReceiptHTML template.HTML
	}{
		ReceiptHTML: template.HTML(string(receipt)),
	}
	if err := tpl.Execute(&out, view); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func buildReceiptPayload(s Sale) map[string]any {
	vatPercent := s.VATPercent
	if vatPercent < 0 {
		vatPercent = 0
	}
	vatIncluded := s.VATIncluded
	vatAmount := roundMoney(s.VATAmount)
	grandTotal := roundMoney(s.TotalAmount)
	afterDiscount := roundMoney(s.SubtotalAmount - s.DiscountAmount)
	if afterDiscount < 0 {
		afterDiscount = 0
	}
	subtotal := roundMoney(s.SubtotalAmount)
	discountBill := roundMoney(s.BillDiscountAmount)
	if discountBill < 0 {
		discountBill = 0
	}
	discountItem := roundMoney(s.DiscountAmount - discountBill)
	if discountItem < 0 {
		discountItem = 0
	}

	items := make([]map[string]any, 0, len(s.Items))
	for _, it := range s.Items {
		code := strings.TrimSpace(it.SKU)
		if code == "" {
			code = it.ProductID
		}
		items = append(items, map[string]any{
			"code":  code,
			"name":  strings.TrimSpace(it.ProductName),
			"qty":   it.Quantity,
			"price": roundMoney(it.UnitPrice),
			"total": roundMoney(it.LineTotal),
		})
	}

	soldAt := s.SoldAt.In(time.Local)
	if soldAt.IsZero() {
		soldAt = time.Now()
	}
	printAt := time.Now().In(time.Local)

	contact := strings.TrimSpace(s.StorePhone)
	if contact == "" {
		contact = "-"
	}
	customerName := strings.TrimSpace(s.CustomerName)
	customerPhone := strings.TrimSpace(s.CustomerPhone)
	staff := strings.TrimSpace(s.CashierName)
	if staff == "" {
		staff = s.CashierUserID
	}
	promptPayID := strings.TrimSpace(s.StorePromptPayID)
	promptPayQR := ""
	if promptPayID != "" {
		promptPayQR = buildPromptPayQRDataURI(promptPayID, grandTotal)
	}

	return map[string]any{
		"store": map[string]any{
			"name":         fallback(s.StoreName, "-"),
			"address":      fallback(s.StoreAddress, "-"),
			"tax_id":       fallback(os.Getenv("APP_STORE_TAX_ID"), "-"),
			"vat_included": vatIncluded,
		},
		"order": map[string]any{
			"order_no":    fallback(s.SaleNumber, s.ID),
			"pos_no":      "",
			"terminal_id": "",
			"staff":       fallback(staff, "-"),
			"datetime":    soldAt.Format("2006-01-02T15:04:05"),
		},
		"customer": map[string]any{
			"name":   customerName,
			"mobile": customerPhone,
		},
		"items": items,
		"summary": map[string]any{
			"subtotal":       subtotal,
			"discount_item":  discountItem,
			"discount_bill":  discountBill,
			"after_discount": afterDiscount,
			"vat_percent":    roundMoney(vatPercent),
			"vat_amount":     vatAmount,
			"grand_total":    grandTotal,
		},
		"payment": map[string]any{
			"received":              roundMoney(s.PaidAmount),
			"change":                roundMoney(s.ChangeAmount),
			"method":                fallback(s.PaymentMethod, "-"),
			"promptpay_id":          promptPayID,
			"promptpay_qr_data_uri": template.URL(promptPayQR),
		},
		"footer": map[string]any{
			"message":    "Thank you.",
			"ref_no":     s.ID,
			"print_date": formatThaiDate(printAt),
			"contact":    contact,
		},
	}
}

func formatMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func formatThaiDate(t time.Time) string {
	return fmt.Sprintf("%d/%d/%d", t.Day(), int(t.Month()), t.Year()+543)
}

func fallback(v, d string) string {
	text := strings.TrimSpace(v)
	if text == "" {
		return d
	}
	return text
}

var nonDigitRegex = regexp.MustCompile(`\D`)

func buildPromptPayQRDataURI(promptPayID string, amount float64) string {
	payload := buildPromptPayPayload(promptPayID, amount)
	if payload == "" {
		return ""
	}

	png, err := qrcode.Encode(payload, qrcode.Medium, 256)
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

func buildPromptPayPayload(promptPayID string, amount float64) string {
	id := normalizePromptPayID(promptPayID)
	if id == "" {
		return ""
	}

	merchantInfo := ""
	switch len(id) {
	case 13:
		if strings.HasPrefix(id, "0066") {
			merchantInfo = formatEMV("29", formatEMV("00", "A000000677010111")+formatEMV("01", id))
		} else {
			merchantInfo = formatEMV("29", formatEMV("00", "A000000677010111")+formatEMV("02", id))
		}
	case 15:
		merchantInfo = formatEMV("29", formatEMV("00", "A000000677010111")+formatEMV("03", id))
	default:
		return ""
	}

	amountValue := ""
	if amount > 0 {
		amountValue = formatEMV("54", fmt.Sprintf("%.2f", amount))
	}

	raw := "000201" + "010211" + merchantInfo + "5802TH" + "5303764" + amountValue + "6304"
	crc := crc16CCITT(raw)
	return raw + strings.ToUpper(fmt.Sprintf("%04X", crc))
}

func normalizePromptPayID(input string) string {
	digits := nonDigitRegex.ReplaceAllString(strings.TrimSpace(input), "")
	if digits == "" {
		return ""
	}

	if len(digits) == 13 && strings.HasPrefix(digits, "0066") {
		return digits
	}
	if len(digits) == 11 && strings.HasPrefix(digits, "66") {
		return "00" + digits
	}
	if len(digits) == 10 && strings.HasPrefix(digits, "0") {
		return "0066" + digits[1:]
	}
	return digits
}

func formatEMV(tag, value string) string {
	return tag + fmt.Sprintf("%02d", len(value)) + value
}

func crc16CCITT(s string) uint16 {
	const poly uint16 = 0x1021
	var crc uint16 = 0xFFFF

	for i := 0; i < len(s); i++ {
		crc ^= uint16(s[i]) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
