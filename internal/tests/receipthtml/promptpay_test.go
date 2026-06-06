package receipthtml_test

import (
	"strings"
	"testing"

	"pos-backend/internal/platform/receipthtml"
)

func TestPromptPayQRDataURI_Valid(t *testing.T) {
	uri := receipthtml.PromptPayQRDataURI("0812345678", 100.0)
	if uri == "" {
		t.Fatal("expected non-empty data URI for valid promptpay id")
	}
	if !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Errorf("expected png data URI, got prefix %q", uri[:min(30, len(uri))])
	}
}

func TestPromptPayQRDataURI_Empty(t *testing.T) {
	if uri := receipthtml.PromptPayQRDataURI("", 100); uri != "" {
		t.Error("empty promptpay id must yield empty URI")
	}
	if uri := receipthtml.PromptPayQRDataURI("abc", 100); uri != "" {
		t.Error("non-numeric promptpay id must yield empty URI")
	}
}

func TestRenderReceipt_QRShownWhenEnabled(t *testing.T) {
	sale := sampleSale()
	sale.PromptPayQRURI = "data:image/png;base64,AAAA"
	cfg := baseCfg()
	cfg.ShowQr = true
	html, err := receipthtml.RenderReceiptHTML(sale, store(), cfg)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, "สแกนเพื่อชำระเงิน (PromptPay)") {
		t.Error("QR block should render when ShowQr=true and URI present")
	}
	if !strings.Contains(out, "data:image/png;base64,AAAA") {
		t.Error("QR image src should be present")
	}
}

func TestRenderReceipt_QRHiddenWhenDisabled(t *testing.T) {
	sale := sampleSale()
	sale.PromptPayQRURI = "data:image/png;base64,AAAA"
	cfg := baseCfg()
	cfg.ShowQr = false // disabled in settings
	html, err := receipthtml.RenderReceiptHTML(sale, store(), cfg)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if strings.Contains(string(html), "สแกนเพื่อชำระเงิน") {
		t.Error("QR block must be hidden when ShowQr=false")
	}
}

func TestRenderReceipt_LogoShownWhenEnabled(t *testing.T) {
	cfg := baseCfg()
	cfg.ShowLogo = true
	cfg.LogoPosition = "top_center"
	st := store()
	st.LogoURL = "https://media.example.com/logo.png"
	html, err := receipthtml.RenderReceiptHTML(sampleSale(), st, cfg)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := string(html)
	if !strings.Contains(out, "https://media.example.com/logo.png") {
		t.Error("logo img src should render when ShowLogo=true and store has LogoURL")
	}
	if !strings.Contains(out, "logo-center") {
		t.Error("logo position class should be applied")
	}
}

func TestRenderReceipt_LogoHiddenWhenNoURL(t *testing.T) {
	cfg := baseCfg()
	cfg.ShowLogo = true
	st := store() // no LogoURL
	html, err := receipthtml.RenderReceiptHTML(sampleSale(), st, cfg)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if strings.Contains(string(html), "<div class=\"logo-wrap") {
		t.Error("logo wrap must be hidden when store has no LogoURL")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
