package dochtml

import (
	"strings"
	"testing"
	"time"
)

func TestQuotationDeliveryTerms_BlankPODateDoesNotChangeDeliveryRule(t *testing.T) {
	days := 15
	price, delivery := quotationTerms(DocData{
		Type:                 "QUOTATION",
		PriceValidityDays:    nil,
		DeliveryLeadTimeDays: &days,
	})
	got := price + " · " + delivery
	if !strings.Contains(got, "ระยะเวลา _ วัน") || got != "ราคานี้ยืนราคาเป็นระยะเวลา _ วัน นับจากวันที่ออกใบเสนอราคา · กำหนดส่งสินค้าภายใน 15 วัน นับจากวันที่ได้รับใบสั่งซื้อ" {
		t.Fatalf("terms = %q, expected date-independent delivery rule", got)
	}
}

func TestQuotationDeliveryTerms_ZeroPODateDoesNotChangeDeliveryRule(t *testing.T) {
	zero := time.Time{}
	_, delivery := quotationTerms(DocData{
		Type:                 "QUOTATION",
		DeliveryLeadTimeDays: func() *int { v := 7; return &v }(),
		POReceivedDate:       &zero,
	})
	if delivery != "กำหนดส่งสินค้าภายใน 7 วัน นับจากวันที่ได้รับใบสั่งซื้อ" {
		t.Fatalf("delivery = %q, expected date-independent delivery rule", delivery)
	}
}
func TestQuotationDeliveryTerms_ZeroDaysUsesUnderscore(t *testing.T) {
	zero := 0
	price, delivery := quotationTerms(DocData{
		Type:                 "QUOTATION",
		PriceValidityDays:    &zero,
		DeliveryLeadTimeDays: &zero,
	})
	got := price + " · " + delivery
	if strings.Contains(got, " 0 วัน") || !strings.Contains(got, "ระยะเวลา _ วัน") || !strings.Contains(got, "ภายใน _ วัน") {
		t.Fatalf("terms = %q, expected zero values to render as underscores", got)
	}
}

func TestDeliveryOrderDateWithoutExplicitDateUsesUnderscore(t *testing.T) {
	got := specialValue(DocData{Type: "DELIVERY_ORDER", DocumentDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)}, "DELIVERY_ORDER")
	if got != "_" {
		t.Fatalf("delivery date = %q, want underscore when not selected", got)
	}
}

func TestQuotationDeliveryTerms_WithPODate(t *testing.T) {
	valid, delivery := 30, 15
	po := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	gotPrice, gotDelivery := quotationTerms(DocData{
		Type:                 "QUOTATION",
		PriceValidityDays:    &valid,
		DeliveryLeadTimeDays: &delivery,
		POReceivedDate:       &po,
	})
	got := gotPrice + " · " + gotDelivery
	if got != "ราคานี้ยืนราคาเป็นระยะเวลา 30 วัน นับจากวันที่ออกใบเสนอราคา · กำหนดส่งสินค้าภายใน 15 วัน นับจากวันที่ได้รับใบสั่งซื้อ" {
		t.Fatalf("terms = %q, expected populated quotation terms", got)
	}
}

func TestQuotationTermsRenderPaymentAndDeliveryAsSeparateRules(t *testing.T) {
	creditDays, deliveryDays := 30, 45
	html, err := RenderUnifiedDocumentHTML(DocData{
		Type:                 "QUOTATION",
		PriceValidityDays:    func() *int { v := 6; return &v }(),
		CreditTermDays:       creditDays,
		DeliveryLeadTimeDays: &deliveryDays,
	}, StoreInfo{})
	if err != nil {
		t.Fatalf("render quotation: %v", err)
	}
	if !strings.Contains(html, "2. เงื่อนไขการชำระเงิน ชำระภายใน 30 วัน นับถัดจากวันที่มอบสินค้าและตรวจรับเรียบร้อยแล้ว") {
		t.Fatal("payment terms rule is missing")
	}
	if !strings.Contains(html, "3. กำหนดส่งสินค้าภายใน 45 วัน นับจากวันที่ได้รับใบสั่งซื้อ") {
		t.Fatal("delivery terms rule is missing")
	}
	if !strings.Contains(html, `<div class="remarks remarks-under-table">`) || !strings.Contains(html, "เงื่อนไขและหมายเหตุ (Terms & Remarks):") {
		t.Fatal("quotation terms are not inside the remarks box below the table")
	}
	if strings.Contains(html, `<section class="termsrow">`) {
		t.Fatal("legacy standalone terms section is still rendered")
	}
	if strings.Contains(html, "หลังจากได้รับใบสั่งซื้อวันที่") {
		t.Fatal("delivery rule still includes a specific PO date")
	}
}
