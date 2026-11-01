package dochtml

import (
	"strings"
	"testing"
	"time"
)

func TestTemplate3DesignKeepsDistinctRenderer(t *testing.T) {
	html, err := RenderTemplate3DocumentHTML(DocData{Type: "QUOTATION", DocumentNoFull: "T3-REAL"}, StoreInfo{Name: "STORE"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyDocumentTemplateDesign(html, "template3", DocumentDesign{PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12, Spacing: "standard", FontFamily: "Sarabun"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `class="page template-template3"`) || !strings.Contains(got, `class="t3-title"`) || strings.Contains(got, `class="page template-classic"`) {
		t.Fatal("template 3 style changed or applied template 2 skin")
	}
}

func TestApplyDocumentTemplateDesignRejectsInvalidConfig(t *testing.T) {
	if _, err := ApplyDocumentTemplateDesign("<html></html>", "unexpected", DocumentDesign{}); err == nil {
		t.Fatal("expected invalid template configuration to return an error")
	}
}

func TestApplyDocumentTemplateDesign(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(DocData{Type: "QUOTATION", DocumentNoFull: "QUO-PREVIEW", DocumentDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)}, StoreInfo{Name: "Store"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyDocumentTemplateDesign(html, "classic", DocumentDesign{PrimaryColor: "#123456", FontSize: 12, Alignment: "center", MarginMM: 15, Spacing: "standard", FontFamily: "Tahoma"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--ink:#123456", "font-family:'Tahoma',sans-serif", "font-size:12px", "text-align:center", "margin:20mm 15mm 16mm", `class="page template-classic"`, `.page.template-classic .items th,.page.template-classic .bill-items th{background:#b6cce3`, `.page.template-classic .doc-info-row{display:grid;grid-template-columns:73fr 27fr`} {
		if !strings.Contains(got, want) {
			t.Errorf("styled HTML does not contain %q", want)
		}
	}
}

func TestStandardTemplateAlwaysKeepsStoreHeader(t *testing.T) {
	store := StoreInfo{Name: "ร้านจริง", Address: "99 ถนนสุขุมวิท", Phone: "02-123-4567", TaxID: "0123456789012", LogoURL: "data:image/png;base64,aGVsbG8="}
	html, err := RenderUnifiedDocumentHTML(DocData{Type: "INVOICE", DocumentNoFull: "INV-TEST"}, store)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyDocumentTemplateDesign(html, "standard", DocumentDesign{PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12, Spacing: "standard", FontFamily: "Sarabun", ShowLogo: false, ShowStoreInfo: false})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ร้านจริง", "99 ถนนสุขุมวิท", "02-123-4567", `class="logo"`, `class="store-name"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing store header %q", want)
		}
	}
	if strings.Contains(got, ".logo{ display:none; }") || strings.Contains(got, ".store-name,.store-meta,.hdr-mini .m-store{ display:none; }") {
		t.Fatal("standard template hid store identity")
	}
}

func TestReferenceTemplateAlwaysKeepsStoreHeader(t *testing.T) {
	store := StoreInfo{Name: "ร้านจริง", Address: "99 ถนนสุขุมวิท", Phone: "02-123-4567", TaxID: "0123456789012", LogoURL: "data:image/png;base64,aGVsbG8="}
	html, err := RenderReferenceDocumentHTML(DocData{Type: "INVOICE", DocumentNoFull: "INV-TEST"}, store)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyDocumentTemplateDesign(html, "classic", DocumentDesign{PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12, Spacing: "standard", FontFamily: "Sarabun", ShowLogo: false, ShowStoreInfo: false})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ร้านจริง", "99 ถนนสุขุมวิท", "02-123-4567", `class="logo"`, `class="ref-store"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing store header %q", want)
		}
	}
	if strings.Contains(got, ".page.template-classic .ref-store{display:none}") || strings.Contains(got, ".page.template-classic .logo{display:none}") {
		t.Fatal("reference template hid store identity despite requiring a complete header")
	}
}

func TestStandardTemplateRemainsAvailableWithoutTemplate2Skin(t *testing.T) {
	html, err := RenderUnifiedDocumentHTML(DocData{Type: "QUOTATION", DocumentNoFull: "QUO-STANDARD"}, StoreInfo{Name: "Store"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyDocumentTemplateDesign(html, "standard", DocumentDesign{PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12, Spacing: "standard", FontFamily: "Sarabun"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, `class="page template-classic"`) || strings.Contains(got, ".page.template-classic .items th") {
		t.Fatal("default standard template was changed by template 2")
	}
	if !strings.Contains(got, "QUO-STANDARD") {
		t.Fatal("standard template document data was lost")
	}
}

func TestTemplate2RendersAllA4DocumentTypes(t *testing.T) {
	for _, docType := range []string{"QUOTATION", "INVOICE", "RECEIPT", "TAX_INVOICE", "BILL", "CREDIT_NOTE", "DELIVERY_ORDER"} {
		t.Run(docType, func(t *testing.T) {
			doc := DocData{Type: docType, DocumentNoFull: "T2-" + docType, Items: []DocItem{{Description: "Sample line", Unit: "ชิ้น", Quantity: 1, UnitPrice: 10, Amount: 10}}, Subtotal: 10, TotalAmount: 10}
			html, err := RenderReferenceDocumentHTML(doc, StoreInfo{Name: "Template Test Store"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := ApplyDocumentTemplateDesign(html, "classic", DocumentDesign{PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12, Spacing: "standard", FontFamily: "Sarabun", ShowLogo: true, ShowStoreInfo: true})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, `class="page template-classic"`) || !strings.Contains(got, "T2-"+docType) || !strings.Contains(got, "background:#b6cce3") || !strings.Contains(got, `class="ref-customer"`) {
				t.Fatal("template 2 did not preserve document data or reference layout")
			}
			if strings.Contains(got, `.page.template-classic .items .col-no{width:8mm}`) {
				t.Fatal("legacy CSS skin corrupted reference table geometry")
			}
		})
	}
}
