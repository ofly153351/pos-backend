package receipt_settings

import "testing"

func TestNormalizeDocumentDesignDefaultsMissingFontFamily(t *testing.T) {
	design := DocumentDesign{PrimaryColor: "#1a1a1a"}
	normalizeDocumentDesign(&design)
	if design.FontFamily != "Sarabun" || design.Spacing != "standard" {
		t.Fatalf("legacy design normalized to font=%q spacing=%q, want Sarabun/standard", design.FontFamily, design.Spacing)
	}
}

func TestValidHexColor(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{{"#123abc", true}, {"#FFFFFF", true}, {"red", false}, {"#12345", false}, {"#12xx56", false}} {
		if got := validHexColor(tc.value); got != tc.want {
			t.Errorf("validHexColor(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestDocumentTemplateKeyValidation(t *testing.T) {
	for _, key := range []string{"standard", "classic", "template3"} {
		if !validDocumentTemplateKey(key) {
			t.Errorf("valid document template %q rejected", key)
		}
	}
	for _, key := range []string{"", "unknown", "../template3"} {
		if validDocumentTemplateKey(key) {
			t.Errorf("invalid document template %q accepted", key)
		}
	}
}

func TestApplyUpdateDocumentTemplateAndDesign(t *testing.T) {
	settings := ReceiptSettings{DocumentTemplateKey: "standard"}
	key := "classic"
	design := DocumentDesign{PrimaryColor: "#123456", FontSize: 12, Alignment: "center", MarginMM: 15}
	applyUpdate(&settings, UpdateReceiptSettingsRequest{DocumentTemplateKey: &key, DocumentDesign: &design})
	if settings.DocumentTemplateKey != key || settings.DocumentDesign != design {
		t.Fatalf("update did not apply: %+v", settings)
	}
}
