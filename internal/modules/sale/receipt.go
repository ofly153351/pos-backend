package sale

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"math"
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
  .right { text-align: right; }
  .muted { color: #606975; }
  .store-name {
    color: #111827;
    font-size: 17px;
    font-weight: 700;
    line-height: 1.25;
    margin-bottom: 4px;
  }
  .store-meta { font-size: 10.5px; line-height: 1.45; }
  .doc-title {
    margin-top: 20px;
    color: #111827;
    font-size: 18px;
    font-weight: 700;
    line-height: 1.2;
  }
  .doc-subtitle {
    margin-top: 2px;
    color: #6b7280;
    font-size: 11px;
  }
  .meta {
    margin-top: 16px;
    display: grid;
    gap: 4px;
  }
  .meta-row {
    display: grid;
    grid-template-columns: 72px 1fr;
    column-gap: 8px;
  }
  .meta-pair {
    display: grid;
    grid-template-columns: 1fr 1fr;
    column-gap: 12px;
  }
  .meta-pair .meta-row:last-child {
    grid-template-columns: 54px 1fr;
  }
  .divider {
    border-top: 1px solid #d9dde3;
    margin: 16px 0 10px;
  }
  table { width: 100%; border-collapse: collapse; }
  th, td { vertical-align: top; }
  th {
    color: #606975;
    font-size: 10.5px;
    font-weight: 600;
    padding: 0 0 7px;
    border-bottom: 1px solid #d9dde3;
  }
  td {
    padding: 6px 0;
    border-bottom: 1px solid #eef0f3;
  }
  .item-name { width: 47%; }
  .qty { width: 14%; text-align: center; }
  .money-col { width: 19.5%; text-align: right; white-space: nowrap; }
  .summary {
    margin-top: 10px;
    display: grid;
    gap: 6px;
  }
  .summary-row {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 12px;
  }
  .summary-total {
    margin-top: 2px;
    padding-top: 8px;
    border-top: 1px solid #d9dde3;
    color: #111827;
    font-size: 13px;
    font-weight: 700;
  }
  .note {
    margin-top: 16px;
    padding: 8px 10px;
    border-radius: 8px;
    background: #f3f4f6;
    color: #374151;
    text-align: center;
    font-size: 11px;
  }
  .qr-block { margin-top: 16px; }
  .qr-block img { width: 150px; height: 150px; }
  @media print {
    @page { margin: 0; size: 80mm auto; }
    body { background: #ffffff; }
    .receipt { width: 80mm; padding: 7mm 6mm; }
  }
</style>
</head>
<body>
<main class="receipt">
  <header class="center">
    <div class="store-name">{{index . "store" "name"}}</div>
    <div class="store-meta muted">
      {{index . "store" "address"}}<br>
      เลขประจำตัวผู้เสียภาษี: {{index . "store" "tax_id"}}<br>
      โทร: {{index . "footer" "contact"}}
    </div>
    <div class="doc-title">ใบเสร็จรับเงิน / ใบกำกับภาษีอย่างย่อ</div>
    <div class="doc-subtitle">(Receipt / Abbreviated Tax Invoice)</div>
  </header>

  <section class="meta">
    <div class="meta-pair">
      <div class="meta-row"><span>เลขที่ (Doc No.):</span><span></span></div>
      <div class="meta-row"><span></span><span class="right">{{index . "order" "order_no"}}</span></div>
    </div>
    <div class="meta-pair">
      <div class="meta-row"><span>วันที่ (Date):</span><span></span></div>
      <div class="meta-row"><span></span><span class="right">{{index . "order" "datetime"}}</span></div>
    </div>
    <div class="meta-pair">
      <div class="meta-row"><span>ลูกค้า (Customer):</span><span></span></div>
      <div class="meta-row"><span></span><span class="right">{{index . "customer" "name"}}</span></div>
    </div>
  </section>

  <div class="divider"></div>

  <table aria-label="receipt items">
    <thead>
      <tr>
        <th class="item-name">รายการ<br>(Item)</th>
        <th class="qty">จำนวน<br>(Qty)</th>
        <th class="money-col">ราคา<br>(Price)</th>
        <th class="money-col">รวม<br>(Total)</th>
      </tr>
    </thead>
    <tbody>
      {{range $item := index . "items"}}
      <tr>
        <td class="item-name">{{index $item "name"}}</td>
        <td class="qty">{{index $item "qty"}}</td>
        <td class="money-col">{{money (index $item "price")}}</td>
        <td class="money-col">{{money (index $item "total")}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>

  <section class="summary">
    {{if index . "summary" "discount_total"}}
    <div class="summary-row muted"><span>ส่วนลด (Discount):</span><span>{{money (index . "summary" "discount_total")}} ฿</span></div>
    {{end}}
    <div class="summary-row"><span>มูลค่าสินค้าก่อนภาษี (Total Before VAT):</span><span>{{money (index . "summary" "total_before_vat")}} ฿</span></div>
    <div class="summary-row"><span>ภาษีมูลค่าเพิ่ม {{index . "summary" "vat_percent"}}% (VAT {{index . "summary" "vat_percent"}}%):</span><span>{{money (index . "summary" "vat_amount")}} ฿</span></div>
    <div class="summary-row summary-total"><span>ยอดชำระสุทธิ (Net Total):</span><span>{{money (index . "summary" "grand_total")}} ฿</span></div>
  </section>

  <div class="note">{{index . "summary" "grand_total_text"}}</div>

  {{if index . "payment" "promptpay_qr_data_uri"}}
  <div class="qr-block center">
    <b>สแกนเพื่อชำระเงิน</b><br>
    <img src="{{index . "payment" "promptpay_qr_data_uri"}}" alt="PromptPay QR"><br>
  </div>
  {{end}}
</main>
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
    min-height: 100vh;
    background: rgba(17, 24, 39, 0.42);
    color: #111827;
    font-family: Arial, Helvetica, sans-serif;
  }
  .toolbar {
    width: 384px;
    margin: 0 auto;
    padding: 0 24px 24px;
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: center;
  }
  .actions { display: flex; gap: 8px; }
  .btn {
    min-width: 132px;
    height: 42px;
    border-radius: 10px;
    border: 0;
    background: #fffdf4;
    color: #374151;
    padding: 0 18px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 700;
  }
  .btn-primary {
    background: #f76f9a;
    color: #ffffff;
    box-shadow: 0 8px 18px rgba(247, 111, 154, 0.34);
  }
  .stage { padding: 28px 12px 0; }
  .paper {
    width: 384px;
    margin: 0 auto;
    background: #ffffff;
    border-radius: 22px;
    box-shadow: 0 18px 42px rgba(15, 23, 42, 0.18);
    overflow: hidden;
  }
  @media print {
    .toolbar, .title { display: none; }
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
  <div class="stage">
    <div class="paper">{{.ReceiptHTML}}</div>
  </div>
  <div class="toolbar">
    <div class="actions">
      <button class="btn btn-primary" type="button" onclick="window.print()">พิมพ์ใบเสร็จ</button>
      <button class="btn" type="button" onclick="window.close()">ปิดหน้าต่าง</button>
    </div>
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
	totalBeforeVAT := afterDiscount
	if vatIncluded {
		totalBeforeVAT = roundMoney(grandTotal - vatAmount)
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
	if customerName == "" {
		customerName = "ลูกค้าทั่วไป (เงินสด)"
	}
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
			"datetime":    formatThaiDateTime(soldAt),
		},
		"customer": map[string]any{
			"name":   customerName,
			"mobile": customerPhone,
		},
		"items": items,
		"summary": map[string]any{
			"subtotal":         subtotal,
			"discount_item":    discountItem,
			"discount_bill":    discountBill,
			"discount_total":   roundMoney(discountItem + discountBill),
			"after_discount":   afterDiscount,
			"total_before_vat": totalBeforeVAT,
			"vat_percent":      roundMoney(vatPercent),
			"vat_amount":       vatAmount,
			"grand_total":      grandTotal,
			"grand_total_text": formatThaiBahtText(grandTotal),
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
		millions := n / 1000000
		remainder := n % 1000000
		text := thaiNumberText(millions) + "ล้าน"
		if remainder > 0 {
			text += thaiNumberText(remainder)
		}
		return text
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
