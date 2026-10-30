package dochtml

import (
	"fmt"
	"regexp"
	"strings"
)

type DocumentDesign struct {
	PrimaryColor  string `json:"primary_color"`
	FontSize      int    `json:"font_size"`
	Alignment     string `json:"alignment"`
	MarginMM      int    `json:"margin_mm"`
	ShowLogo      bool   `json:"show_logo"`
	ShowStoreInfo bool   `json:"show_store_info"`
	Spacing       string `json:"spacing"`
	FontFamily    string `json:"font_family"`
}

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// ApplyDocumentTemplateDesign applies validated store-level presentation options
// to the shared A4 renderer output. Invalid persisted values fail loudly.
func ApplyDocumentTemplateDesign(html, templateKey string, design DocumentDesign) (string, error) {
	if templateKey != "standard" && templateKey != "classic" {
		return "", fmt.Errorf("unsupported document template %q", templateKey)
	}
	if !hexColorPattern.MatchString(design.PrimaryColor) || design.FontSize < 8 || design.FontSize > 16 || design.MarginMM < 5 || design.MarginMM > 30 || (design.Spacing != "compact" && design.Spacing != "standard" && design.Spacing != "comfortable") || (design.FontFamily != "Sarabun" && design.FontFamily != "Tahoma" && design.FontFamily != "Arial") || (design.Alignment != "left" && design.Alignment != "center" && design.Alignment != "right") {
		return "", fmt.Errorf("invalid document design")
	}
	family := "'Sarabun','Tahoma',sans-serif"
	if design.FontFamily == "Tahoma" {
		family = "'Tahoma',sans-serif"
	}
	if design.FontFamily == "Arial" {
		family = "'Arial',sans-serif"
	}
	html = strings.Replace(html, "'Sarabun','Tahoma',sans-serif", family, 1)
	html = strings.Replace(html, "--ink:#1a1a1a", "--ink:"+design.PrimaryColor, 1)
	html = strings.Replace(html, "font-size:11px; background:#fff", fmt.Sprintf("font-size:%dpx; background:#fff", design.FontSize), 1)
	html = strings.Replace(html, "-webkit-print-color-adjust:exact", "text-align:"+design.Alignment+"; -webkit-print-color-adjust:exact", 1)
	html = strings.Replace(html, "margin:20mm 12mm 16mm", fmt.Sprintf("margin:20mm %dmm 16mm", design.MarginMM), 1)
	if design.Spacing == "compact" {
		html = strings.Replace(html, "--row-h:7.5mm;", "--row-h:6mm;", 1)
	} else if design.Spacing == "comfortable" {
		html = strings.Replace(html, "--row-h:7.5mm;", "--row-h:10mm;", 1)
	}
	// Store identity is mandatory in the A4 document header for both templates.
	// Older settings may explicitly contain false; keep the fields for backward
	// compatibility, but do not turn them into CSS that erases the header.
	if templateKey == "classic" {
		html = strings.ReplaceAll(html, `class="page"`, `class="page template-classic"`)
		// Legacy classic skin is for the unified DOM; the reference renderer owns
		// its own geometry and must not inherit the old fixed column widths.
		if !strings.Contains(html, `class="ref-header"`) {
			html = strings.Replace(html, "</style>", quotationTemplate2CSS+"</style>", 1)
		}
	}
	return html, nil
}

// quotationTemplate2CSS carries the visual language of the user-provided
// quotation-recreated.html into the existing data-driven A4 layout. Keeping the
// shared DOM preserves live document fields, copy handling, and Paged.js flow.
const quotationTemplate2CSS = `
.page.template-classic .hdr{padding-bottom:4mm;border-bottom:2px solid #080808}
.page.template-classic .hdr-left{gap:3mm}
.page.template-classic .store-name{font-size:20px;letter-spacing:.3px}
.page.template-classic .store-meta{font-size:9px;line-height:1.3;color:#111}
.page.template-classic .doc-title{font-size:22px;line-height:1.1;color:#080808}
.page.template-classic .doc-id,.page.template-classic .doc-badge{border:1.2px solid #080808;border-radius:0}
.page.template-classic .doc-info-row{display:grid;grid-template-columns:73fr 27fr;gap:0;margin:2mm 0 3mm;border:2px solid #080808;break-inside:avoid;page-break-inside:avoid}
.page.template-classic .parties{border:0;min-width:0}
.page.template-classic .party{padding:0 3mm 2mm}
.page.template-classic .doc-meta-panel{border:0;border-left:1.5px solid #080808;padding:0 2mm 2mm}
.page.template-classic .header-section-title{margin:0 -3mm 2mm;padding:1.4mm 3mm;background:#b6cce3;color:#080808;border-bottom:1.5px solid #080808;font-size:10px;line-height:1.3}
.page.template-classic .doc-meta-panel .header-section-title{margin-left:-2mm;margin-right:-2mm}
.page.template-classic .doc-meta-label,.page.template-classic .info-label{color:#222}
.page.template-classic .doc-meta-value,.page.template-classic .info-value{color:#080808}
.page.template-classic .quotation-intro{font-size:11px;color:#111;margin:0 0 3mm}
.page.template-classic .items,.page.template-classic .bill-items{border:2px solid #080808}
.page.template-classic .items th,.page.template-classic .bill-items th{background:#b6cce3;color:#080808;border:1px solid #080808;padding:1.6mm 1mm;font-size:10px;font-weight:500}
.page.template-classic .items td,.page.template-classic .bill-items td{border-color:#080808;padding:1.2mm 1mm;font-size:10px}
.page.template-classic .items .col-no{width:8mm}
.page.template-classic .items .col-qty{width:16mm}
.page.template-classic .items .col-unit{width:13mm}
.page.template-classic .items .col-price{width:23mm}
.page.template-classic .items .col-disc{width:19mm}
.page.template-classic .items .col-amt{width:24mm}
.page.template-classic .items .desc{line-height:1.45}
.page.template-classic .baht-text-row{border:1.5px solid #080808;background:#fff}
.page.template-classic .remarks-under-table .remarks,.page.template-classic .remarks{border:1.5px solid #080808;border-radius:0;padding:2mm 3mm}
.page.template-classic .sum-row{border-bottom:1px solid #080808;padding:1.2mm 2mm}
.page.template-classic .sum-total{border-top:1.5px solid #080808;border-bottom:1.5px solid #080808;font-size:14px}
.page.template-classic .sig-t{border-bottom:1px solid #080808}
.page.template-classic .sig-ln,.page.template-classic .sig-date-seg{border-color:#080808}
@media screen{.page.template-classic{box-shadow:0 2px 8px rgba(15,23,42,.16)}}
`
