package dochtml

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"pos-backend/internal/platform/htmlpdf"
)

func TestTemplate2PagedPDFKeepsMultiPageDocumentFlow(t *testing.T) {
	items := make([]DocItem, 80)
	for i := range items {
		items[i] = DocItem{
			Description: fmt.Sprintf("รายการทดสอบ Template 2 หมายเลข %02d พร้อมรายละเอียดภาษาไทยเพื่อทดสอบการแบ่งหน้า", i+1),
			Unit:        "ชิ้น",
			Quantity:    1,
			UnitPrice:   10,
			Amount:      10,
		}
	}
	html, err := RenderUnifiedDocumentHTML(DocData{
		Type:           "QUOTATION",
		DocumentNoFull: "T2-PAGED-TEST",
		DocumentDate:   time.Date(2026, 10, 30, 0, 0, 0, 0, time.FixedZone("ICT", 7*60*60)),
		Items:          items,
		Subtotal:       800,
		TotalAmount:    800,
	}, StoreInfo{Name: "Template 2 Test Store"})
	if err != nil {
		t.Fatalf("render source HTML: %v", err)
	}
	html, err = ApplyDocumentTemplateDesign(html, "classic", DocumentDesign{
		PrimaryColor: "#1a1a1a", FontSize: 11, Alignment: "left", MarginMM: 12,
		Spacing: "standard", FontFamily: "Sarabun", ShowLogo: true, ShowStoreInfo: true,
	})
	if err != nil {
		t.Fatalf("apply template 2: %v", err)
	}
	pdf, err := htmlpdf.Render(context.Background(), html)
	if err != nil {
		t.Fatalf("render template 2 with Paged.js: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", pdf[:min(len(pdf), 8)])
	}
	pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pages < 2 {
		t.Fatalf("expected a multi-page template 2 PDF, got %d pages", pages)
	}
}
