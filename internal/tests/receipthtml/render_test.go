package receipthtml_test

import (
	"strings"
	"testing"

	"pos-backend/internal/platform/receipthtml"
)

func sampleSale() receipthtml.SaleData {
	return receipthtml.SaleData{
		OrderNo:        "SALE-20260601-0001",
		DateTime:       "01/06/2569 11:23",
		CustomerName:   "ทั่วไป",
		Staff:          "แคชเชียร์ 1",
		PaymentMethod:  "cash",
		Items: []receipthtml.SaleItem{
			{Name: "แก๊ส 15kg", Qty: 1, Price: 350, Total: 350},
			{Name: "โค้ก 1.25L", Qty: 2, Price: 35, Total: 70},
		},
		Subtotal:       420,
		AfterDiscount:  420,
		VatAmount:      0,
		GrandTotal:     420,
		GrandTotalText: "สี่ร้อยยี่สิบบาทถ้วน",
		Paid:           500,
		Change:         80,
	}
}

func baseCfg() receipthtml.Config {
	return receipthtml.Config{
		ShowStoreName: true,
		ShowAddress:   true,
		ShowPhone:     true,
		ShowTaxId:     true,
		TaxMode:       "none",
		VatRate:       7,
		TaxLabel:      "VAT",
		PaperSize:     "80mm",
	}
}

func store() receipthtml.StoreInfo {
	return receipthtml.StoreInfo{Name: "POS Demo Store", Address: "123 ถนนสุขุมวิท", Phone: "02-000-0000", TaxID: "-"}
}

func TestRenderReceipt_NewLayout(t *testing.T) {
	html, err := receipthtml.RenderReceiptHTML(sampleSale(), store(), baseCfg())
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := string(html)

	mustContain := []string{
		"POS Demo Store",
		"ใบเสร็จรับเงิน / ใบกำกับภาษีอย่างย่อ",
		"Receipt / Abbreviated Tax Invoice",
		"1. แก๊ส 15kg",     // numbered item
		"2. โค้ก 1.25L",
		"1 x 350.00",       // qty x price line
		"2 x 35.00",
		"รายการทั้งหมด 2 รายการ", // item count
		"3 ชิ้น",            // total qty (1+2)
		"ยอดสุทธิ",
		"฿420.00",          // grand total with baht symbol
		"รับเงิน",
		"฿500.00",          // paid with baht symbol
		"เงินทอน",
		"฿80.00",           // change with baht symbol
		"size:80mm auto",   // print paper size
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("receipt HTML missing %q", s)
		}
	}
}

func TestRenderReceipt_BahtThousandsSeparator(t *testing.T) {
	sale := sampleSale()
	sale.GrandTotal = 1200
	sale.Paid = 1500
	sale.Change = 300
	html, err := receipthtml.RenderReceiptHTML(sale, store(), baseCfg())
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := string(html)
	for _, want := range []string{"฿1,200.00", "฿1,500.00", "฿300.00"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in receipt (baht symbol + comma separator)", want)
		}
	}
}

func TestRenderReceipt_ExclusiveVAT(t *testing.T) {
	cfg := baseCfg()
	cfg.TaxMode = "exclusive"
	sale := sampleSale()
	sale.AfterDiscount = 392.52
	sale.VatAmount = 27.48
	html, err := receipthtml.RenderReceiptHTML(sale, store(), cfg)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, "รวมก่อน VAT") {
		t.Error("exclusive mode should show 'รวมก่อน VAT'")
	}
	if !strings.Contains(out, "VAT 7%") {
		t.Error("exclusive mode should show 'VAT 7%'")
	}
}

func TestRenderReceipt_TotalQtyAutofill(t *testing.T) {
	sale := sampleSale()
	sale.TotalQty = 0 // force autofill
	html, err := receipthtml.RenderReceiptHTML(sale, store(), baseCfg())
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(string(html), "3 ชิ้น") {
		t.Error("TotalQty autofill should compute 3 from items 1+2")
	}
}

func TestRenderReceipt_PaymentLabelAutofill(t *testing.T) {
	sale := sampleSale()
	sale.PaymentLabel = "" // force autofill from PaymentMethod="cash"
	html, err := receipthtml.RenderReceiptHTML(sale, store(), baseCfg())
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(string(html), "เงินสด") {
		t.Error("PaymentLabel autofill should map cash → เงินสด")
	}
}

func TestPaymentLabel(t *testing.T) {
	cases := map[string]string{
		"cash":      "เงินสด",
		"promptpay": "พร้อมเพย์ / โอน",
		"card":      "บัตรเครดิต / เดบิต",
		"":          "-",
		"weird":     "weird",
	}
	for in, want := range cases {
		if got := receipthtml.PaymentLabel(in); got != want {
			t.Errorf("PaymentLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderReceipt_HidesPaidWhenZero(t *testing.T) {
	sale := sampleSale()
	sale.Paid = 0
	sale.Change = 0
	html, err := receipthtml.RenderReceiptHTML(sale, store(), baseCfg())
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	// ">รับเงิน<" is the row label; plain "รับเงิน" also appears inside the title "ใบเสร็จรับเงิน"
	if strings.Contains(string(html), ">รับเงิน<") {
		t.Error("should hide 'รับเงิน' row when Paid is zero")
	}
}
