package sale

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"
)

func TestReceiptTemplateRendersAbbreviatedTaxInvoiceLayout(t *testing.T) {
	soldAt := time.Date(2026, 3, 28, 15, 31, 0, 0, time.Local)
	payload := buildReceiptPayload(Sale{
		ID:             "sale-1",
		StoreName:      "บริษัท ร้านค้าทดสอบ POS จำกัด",
		StoreAddress:   "123 ถนนสุขุมวิท แขวงคลองเตบ เขตคลองเตย กทม 10110",
		StorePhone:     "1234567890123",
		SaleNumber:     "INV-271639",
		CustomerName:   "ลูกค้าทั่วไป (เงินสด)",
		SubtotalAmount: 195,
		VATIncluded:    true,
		VATPercent:     7,
		VATAmount:      12.76,
		TotalAmount:    195,
		PaidAmount:     195,
		SoldAt:         soldAt,
		Items: []SaleItem{
			{ProductName: "แซนด์วิช (Sandwich)", Quantity: 3, UnitPrice: 40, LineTotal: 120},
			{ProductName: "คุกกี้ (Cookies)", Quantity: 1, UnitPrice: 30, LineTotal: 30},
			{ProductName: "ชาเขียว (Green Tea)", Quantity: 1, UnitPrice: 45, LineTotal: 45},
		},
	})

	tpl, err := template.New("receipt").Funcs(template.FuncMap{
		"money": func(v any) string {
			n, _ := v.(float64)
			return formatMoney(n)
		},
	}).Parse(receiptTemplate)
	if err != nil {
		t.Fatalf("parse receipt template: %v", err)
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, payload); err != nil {
		t.Fatalf("execute receipt template: %v", err)
	}

	html := out.String()
	for _, want := range []string{
		"ใบเสร็จรับเงิน / ใบกำกับภาษีอย่างย่อ",
		"INV-271639",
		"28/3/2569 15:31",
		"แซนด์วิช (Sandwich)",
		"182.24 ฿",
		"12.76 ฿",
		"195.00 ฿",
		"หนึ่งร้อยเก้าสิบห้าบาทถ้วน",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered receipt missing %q", want)
		}
	}
}

func TestFormatThaiBahtText(t *testing.T) {
	tests := map[float64]string{
		0:       "ศูนย์บาทถ้วน",
		11:      "สิบเอ็ดบาทถ้วน",
		21:      "ยี่สิบเอ็ดบาทถ้วน",
		101:     "หนึ่งร้อยเอ็ดบาทถ้วน",
		1300:    "หนึ่งพันสามร้อยบาทถ้วน",
		1300.50: "หนึ่งพันสามร้อยบาทห้าสิบสตางค์",
	}

	for input, want := range tests {
		if got := formatThaiBahtText(input); got != want {
			t.Fatalf("formatThaiBahtText(%v) = %q, want %q", input, got, want)
		}
	}
}
