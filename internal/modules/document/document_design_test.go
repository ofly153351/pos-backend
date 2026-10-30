package document

import (
	"encoding/json"
	"testing"

	"pos-backend/internal/platform/dochtml"
)

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
