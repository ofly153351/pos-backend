package dochtml

import (
	"strings"
	"testing"
	"time"
)

func TestQuotationDeliveryTerms_BlankPODateUsesUnderscore(t *testing.T) {
	days := 15
	price, delivery := quotationTerms(DocData{
		Type:                 "QUOTATION",
		PriceValidityDays:    nil,
		DeliveryLeadTimeDays: &days,
	})
	got := price + " · " + delivery
	if !strings.Contains(got, "ระยะเวลา _ วัน") || !strings.Contains(got, "ภายใน 15 วัน") || !strings.Contains(got, "วันที่ _") {
		t.Fatalf("terms = %q, expected underscores for blank values", got)
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
	if !strings.Contains(got, "ระยะเวลา 30 วัน") || !strings.Contains(got, "ภายใน 15 วัน") || strings.Contains(got, "วันที่ _") {
		t.Fatalf("terms = %q, expected populated quotation terms", got)
	}
}
