package document

import (
	"encoding/json"
	"testing"

	"pos-backend/internal/platform/dochtml"
)

func TestResolveDocumentTemplatePreviewOverride(t *testing.T) {
	for _, tc := range []struct{ saved, preview, want string }{
		{"classic", "standard", "standard"},
		{"standard", "classic", "classic"},
		{"standard", "template3", "template3"},
		{"template3", "standard", "standard"},
		{"template3", "", "template3"},
		{"classic", "", "classic"},
		{"", "", "standard"},
	} {
		got, err := resolveDocumentTemplate(tc.saved, tc.preview)
		if err != nil || got != tc.want {
			t.Fatalf("saved=%q preview=%q: got %q, %v; want %q", tc.saved, tc.preview, got, err, tc.want)
		}
	}
	if _, err := resolveDocumentTemplate("standard", "unknown"); err == nil {
		t.Fatal("unknown preview template must be rejected without changing saved settings")
	}
}

func TestNormalizeDocumentDesignLegacySettings(t *testing.T) {
	raw := json.RawMessage(`{"primary_color":"#1a1a1a","font_size":11,"alignment":"left","margin_mm":12}`)
	var design dochtml.DocumentDesign
	if err := json.Unmarshal(raw, &design); err != nil {
		t.Fatal(err)
	}
	if err := normalizeDocumentDesignForPrint(raw, &design); err != nil {
		t.Fatal(err)
	}
	if design.FontFamily != "Sarabun" || design.Spacing != "standard" || !design.ShowLogo || !design.ShowStoreInfo {
		t.Fatalf("legacy design defaults = %+v", design)
	}
	if _, err := dochtml.ApplyDocumentTemplateDesign("<html></html>", "standard", design); err != nil {
		t.Fatalf("legacy settings should render: %v", err)
	}
}

func TestNormalizeDocumentDesignEmptyPersistedFontFamily(t *testing.T) {
	raw := json.RawMessage(`{"spacing":"standard","alignment":"left","font_size":11,"margin_mm":12,"show_logo":true,"font_family":"","primary_color":"#1a1a1a","show_store_info":true}`)
	var design dochtml.DocumentDesign
	if err := json.Unmarshal(raw, &design); err != nil {
		t.Fatal(err)
	}
	if err := normalizeDocumentDesignForPrint(raw, &design); err != nil {
		t.Fatal(err)
	}
	if design.FontFamily != "Sarabun" {
		t.Fatalf("empty persisted font family normalized to %q, want Sarabun", design.FontFamily)
	}
	for _, key := range []string{"standard", "classic", "template3"} {
		if _, err := dochtml.ApplyDocumentTemplateDesign("<html></html>", key, design); err != nil {
			t.Errorf("template %s rejected legacy design: %v", key, err)
		}
	}
}

func TestNormalizeDocumentDesignPreservesExplicitFalse(t *testing.T) {
	raw := json.RawMessage(`{"primary_color":"#1a1a1a","font_size":11,"alignment":"left","margin_mm":12,"show_logo":false,"show_store_info":false}`)
	var design dochtml.DocumentDesign
	if err := json.Unmarshal(raw, &design); err != nil {
		t.Fatal(err)
	}
	if err := normalizeDocumentDesignForPrint(raw, &design); err != nil {
		t.Fatal(err)
	}
	if design.ShowLogo || design.ShowStoreInfo {
		t.Fatalf("explicit false values were not preserved: %+v", design)
	}
}
