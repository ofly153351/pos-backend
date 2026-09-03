package sale

import (
	"strings"
	"testing"
	"time"
)

// C-01 regression: the sale's VAT snapshot must be resolved from the store's
// receipt settings at checkout, with the request carrying only the cashier's
// per-bill on/off intent (zero = off, non-zero = on).
func TestEffectiveTaxPolicy(t *testing.T) {
	seven := 7.0
	zero := 0.0
	cases := []struct {
		name       string
		settings   ReceiptSettingsView
		reqPercent *float64
		wantInc    bool
		wantPct    float64
	}{
		{
			name:       "QA case: exclusive 7% settings, frontend sends vat_percent 0 (toggle off)",
			settings:   ReceiptSettingsView{TaxMode: "exclusive", VatRate: 7},
			reqPercent: &zero,
			wantInc:    false,
			wantPct:    0,
		},
		{
			name:       "exclusive 7% settings, VAT toggled on → rate comes from settings",
			settings:   ReceiptSettingsView{TaxMode: "exclusive", VatRate: 7},
			reqPercent: &seven,
			wantInc:    false,
			wantPct:    7,
		},
		{
			name:       "nil vat_percent (bare API sale) keeps the settings default: VAT on",
			settings:   ReceiptSettingsView{TaxMode: "exclusive", VatRate: 7},
			reqPercent: nil,
			wantInc:    false,
			wantPct:    7,
		},
		{
			name:       "inclusive settings → VAT-inclusive snapshot when toggled on",
			settings:   ReceiptSettingsView{TaxMode: "inclusive", VatRate: 7},
			reqPercent: &seven,
			wantInc:    true,
			wantPct:    7,
		},
		{
			name:       "tax_mode none forces VAT off even when the toggle is on",
			settings:   ReceiptSettingsView{TaxMode: "none", VatRate: 7},
			reqPercent: &seven,
			wantInc:    false,
			wantPct:    0,
		},
		{
			name:       "unknown tax_mode falls back to exclusive",
			settings:   ReceiptSettingsView{TaxMode: "bogus", VatRate: 7},
			reqPercent: &seven,
			wantInc:    false,
			wantPct:    7,
		},
		{
			name:       "client cannot smuggle a custom rate: vat_percent 999 still applies the settings rate",
			settings:   ReceiptSettingsView{TaxMode: "exclusive", VatRate: 7},
			reqPercent: func() *float64 { v := 999.0; return &v }(),
			wantInc:    false,
			wantPct:    7,
		},
		{
			name:       "negative vat_percent is clamped by validation upstream; policy treats it as off",
			settings:   ReceiptSettingsView{TaxMode: "exclusive", VatRate: 7},
			reqPercent: func() *float64 { v := -1.0; return &v }(),
			wantInc:    false,
			wantPct:    0,
		},
		{
			name:       "settings rate zero keeps VAT off even when toggled on",
			settings:   ReceiptSettingsView{TaxMode: "inclusive", VatRate: 0},
			reqPercent: &seven,
			wantInc:    false,
			wantPct:    0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotInc, gotPct := effectiveTaxPolicy(tc.settings, tc.reqPercent)
			if gotInc != tc.wantInc || gotPct != tc.wantPct {
				t.Fatalf("effectiveTaxPolicy(%+v, %v) = (%v, %v), want (%v, %v)",
					tc.settings, tc.reqPercent, gotInc, gotPct, tc.wantInc, tc.wantPct)
			}
		})
	}
}

// C-01 timestamp half: the receipt must format the sale time in Asia/Bangkok,
// never in the server's local zone. Simulate the prod host running UTC-4 (the
// zone that produced QA's 04:48 vs 15:48 divergence) and prove the receipt
// prints Bangkok wall time regardless.
func TestReceiptUsesBangkokTime(t *testing.T) {
	// 2026-09-02 08:48:59 UTC == 15:48 Bangkok (the QA evidence figure).
	soldAt := time.Date(2026, 9, 2, 8, 48, 59, 0, time.UTC)

	origLocal := time.Local
	time.Local = time.FixedZone("EDT", -4*60*60) // simulate the prod host (UTC-4)
	defer func() { time.Local = origLocal }()

	html, err := renderSaleAsReceiptHTML(Sale{
		SaleNumber:      "SALE-TEST",
		SoldAt:          soldAt,
		PaymentMethod:   "cash",
		PaidAmount:      214,
		TotalAmount:     214,
		SubtotalAmount:  200,
		VATIncluded:     false,
		VATPercent:      7,
		VATAmount:       14,
		Items:           []SaleItem{{ProductName: "ทดสอบ", Quantity: 2, UnitPrice: 107, LineTotal: 214}},
	}, defaultSettings())
	if err != nil {
		t.Fatalf("renderSaleAsReceiptHTML: %v", err)
	}
	s := string(html)
	if !strings.Contains(s, "15:48") {
		t.Fatalf("receipt DateTime must print Asia/Bangkok wall time 15:48, got:\n%.400s", s)
	}
	if strings.Contains(s, "04:48") {
		t.Fatalf("receipt DateTime must not print server-local (UTC-4) time 04:48 — the exact C-01 symptom")
	}
}

// C-01 receipt-title half: the printed title must reflect whether the sale
// ACTUALLY carried VAT. QA's invalid tax receipt = a zero-VAT sale that still
// printed "ใบกำกับภาษีอย่างย่อ" (abbreviated tax invoice). The title shares the
// same gate as the VAT rows: VatAmount > 0.
func TestReceiptTitleMatchesVAT(t *testing.T) {
	base := Sale{
		SaleNumber:    "SALE-TITLE",
		PaymentMethod: "cash",
		PaidAmount:    214,
		TotalAmount:   214,
		Items:         []SaleItem{{ProductName: "ทดสอบ", Quantity: 1, UnitPrice: 214, LineTotal: 214}},
	}

	// Sale that carried 7% exclusive VAT: title must claim the abbreviated tax invoice.
	vatSale := base
	vatSale.SubtotalAmount = 200
	vatSale.VATIncluded = false
	vatSale.VATPercent = 7
	vatSale.VATAmount = 14
	// zero-VAT sale (the QA case): title must be a plain receipt.
	noVatSale := base
	noVatSale.SubtotalAmount = 214

	withVAT, err := renderSaleAsReceiptHTML(vatSale, defaultSettings())
	if err != nil {
		t.Fatalf("renderSaleAsReceiptHTML(vatSale): %v", err)
	}
	sWith := string(withVAT)
	if !strings.Contains(sWith, "ใบเสร็จรับเงิน / ใบกำกับภาษีอย่างย่อ") ||
		!strings.Contains(sWith, "Receipt / Abbreviated Tax Invoice") {
		t.Fatalf("VAT receipt must print the abbreviated-tax-invoice title, got:\n%.300s", sWith)
	}

	withoutVAT, err := renderSaleAsReceiptHTML(noVatSale, defaultSettings())
	if err != nil {
		t.Fatalf("renderSaleAsReceiptHTML(noVatSale): %v", err)
	}
	sWithout := string(withoutVAT)
	if strings.Contains(sWithout, "ใบกำกับภาษี") {
		t.Fatalf("zero-VAT receipt must NOT claim to be a tax invoice (QA invalid-tax-receipt defect), got:\n%.300s", sWithout)
	}
	if !strings.Contains(sWithout, `<div class="doc-title">ใบเสร็จรับเงิน</div>`) ||
		!strings.Contains(sWithout, `<div class="doc-title-en">Receipt</div>`) {
		t.Fatalf("zero-VAT receipt must print the plain-receipt title, got:\n%.300s", sWithout)
	}
}
