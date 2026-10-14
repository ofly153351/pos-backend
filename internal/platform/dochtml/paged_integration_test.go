package dochtml

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"pos-backend/internal/platform/htmlpdf"
)

func TestPagedPDFStressKeepsLargeDocumentContinuous(t *testing.T) {
	items := make([]DocItem, 200)
	for i := range items {
		items[i] = DocItem{
			Description:   fmt.Sprintf("สินค้าทดสอบรายการที่ %03d ชื่อภาษาไทยยาวเพื่อทดสอบการตัดบรรทัดและความสูงของแถว", i+1),
			DescriptionEn: "Long product description for variable row-height pagination testing",
			SKU:           fmt.Sprintf("SKU-%03d", i+1),
			Unit:          "ชิ้น",
			Quantity:      1,
			UnitPrice:     100,
			Amount:        100,
		}
	}

	html, err := RenderUnifiedDocumentHTML(DocData{
		Type:            "INVOICE",
		DocumentNoFull:  "INV-STRESS-200",
		DocumentDate:    time.Date(2026, 10, 6, 0, 0, 0, 0, time.FixedZone("ICT", 7*60*60)),
		CustomerName:    "ลูกค้าทดสอบเอกสารหลายหน้า",
		CustomerAddress: "ที่อยู่ทดสอบสำหรับเอกสาร PDF หลายหน้า",
		Items:           items,
		Subtotal:        20000,
		VatRate:         7,
		VatAmount:       1400,
		TotalAmount:     21400,
	}, StoreInfo{Name: "ร้านทดสอบ"})
	if err != nil {
		t.Fatalf("render HTML: %v", err)
	}

	pdf, err := htmlpdf.Render(context.Background(), html)
	if err != nil {
		t.Fatalf("render Paged.js PDF: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", pdf[:min(len(pdf), 8)])
	}
	pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pages < 5 {
		t.Fatalf("200 products should produce at least 5 natural pages, got %d", pages)
	}
	if path := os.Getenv("PAGED_PDF_OUT"); path != "" {
		if err := os.WriteFile(path, pdf, 0o600); err != nil {
			t.Fatalf("write debug PDF: %v", err)
		}
	}
	t.Logf("Paged.js produced %d PDF pages for 200 products", pages)
	if bytes.Contains(pdf, []byte("footer-only")) || bytes.Contains(pdf, []byte("filler")) {
		t.Fatal("PDF must not contain manual footer-only/filler pagination markers")
	}
}
