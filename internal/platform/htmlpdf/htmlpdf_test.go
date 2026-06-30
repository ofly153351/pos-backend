package htmlpdf

import (
	"bytes"
	"context"
	"testing"
)

func TestRender_MultiPage(t *testing.T) {
	// Two A4 sheets separated by a forced page break.
	html := `<!DOCTYPE html><html><head><meta charset="utf-8">
<style>@page{size:A4;margin:0} .s{height:297mm;background:#eee}</style></head>
<body><div class="s">หน้า 1 ต้นฉบับ (Original)</div>
<div class="s" style="page-break-before:always">หน้า 2 สำเนา (Copy)</div></body></html>`

	pdf, err := Render(context.Background(), html)
	if err != nil {
		t.Fatalf("render (Chrome installed?): %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", pdf[:min(8, len(pdf))])
	}
	// A two-sheet document must span multiple pages (exact count varies by Chrome's
	// PDF object layout, so assert >= 2 rather than an exact number).
	pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pages < 2 {
		t.Fatalf("want >= 2 pages, got %d", pages)
	}
	if len(pdf) < 1000 {
		t.Fatalf("PDF suspiciously small: %d bytes", len(pdf))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
