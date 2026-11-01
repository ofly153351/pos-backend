package document

import (
	"encoding/json"
	"fmt"

	"pos-backend/internal/platform/dochtml"
)

func resolveDocumentTemplate(saved, preview string) (string, error) {
	if preview != "" {
		if preview != "standard" && preview != "classic" && preview != "template3" {
			return "", fmt.Errorf("unknown document template: %w", ErrInvalidInput)
		}
		return preview, nil
	}
	if saved == "classic" || saved == "template3" {
		return saved, nil
	}
	return "standard", nil
}

// normalizeDocumentDesignForPrint fills fields omitted by older persisted JSONB
// settings while preserving explicitly stored values (including false booleans).
func normalizeDocumentDesignForPrint(raw json.RawMessage, design *dochtml.DocumentDesign) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if _, ok := fields["primary_color"]; !ok {
		design.PrimaryColor = "#1a1a1a"
	}
	if _, ok := fields["font_size"]; !ok {
		design.FontSize = 11
	}
	if _, ok := fields["alignment"]; !ok {
		design.Alignment = "left"
	}
	if _, ok := fields["margin_mm"]; !ok {
		design.MarginMM = 12
	}
	if _, ok := fields["show_logo"]; !ok {
		design.ShowLogo = true
	}
	if _, ok := fields["show_store_info"]; !ok {
		design.ShowStoreInfo = true
	}
	if _, ok := fields["spacing"]; !ok {
		design.Spacing = "standard"
	}
	if _, ok := fields["font_family"]; !ok {
		design.FontFamily = "Sarabun"
	}
	return nil
}
