package docpdf

import (
	_ "embed"
	"os"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

//go:embed assets/Tahoma.ttf
var pdfFontTahoma []byte

//go:embed assets/DejaVuSansCondensed.ttf
var pdfFontDejaVu []byte

// registerFont adds the best available Thai-compatible UTF-8 font and returns
// the family name to pass to SetFont.
func registerFont(pdf *gofpdf.Fpdf) string {
	if len(pdfFontTahoma) > 0 {
		pdf.AddUTF8FontFromBytes("TH", "", pdfFontTahoma)
		if pdf.Ok() {
			return "TH"
		}
		pdf.ClearError()
	}
	if len(pdfFontDejaVu) > 0 {
		pdf.AddUTF8FontFromBytes("TH", "", pdfFontDejaVu)
		if pdf.Ok() {
			return "TH"
		}
		pdf.ClearError()
	}
	// environment override
	if path := strings.TrimSpace(os.Getenv("PDF_TTF_PATH")); path != "" {
		if _, err := os.Stat(path); err == nil {
			pdf.AddUTF8Font("TH", "", path)
			if pdf.Ok() {
				return "TH"
			}
			pdf.ClearError()
		}
	}
	// OS fallbacks
	for _, p := range []string{
		"/System/Library/Fonts/Supplemental/Tahoma.ttf",
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/usr/share/fonts/truetype/noto/NotoSansThai-Regular.ttf",
	} {
		if _, err := os.Stat(p); err == nil {
			pdf.AddUTF8Font("TH", "", p)
			if pdf.Ok() {
				return "TH"
			}
			pdf.ClearError()
		}
	}
	return "Arial"
}
