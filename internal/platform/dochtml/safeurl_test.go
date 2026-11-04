package dochtml

import (
	"strings"
	"testing"
)

// B29: user-controlled values must render as text and logo URLs must not
// carry script URLs into previews.
func TestPreviewInjectionIsNeutralised(t *testing.T) {
	for _, bad := range []string{"javascript:alert(1)", "JaVaScRiPt:alert(1)", "data:text/html,<script>x</script>", `https://x/"onerror="alert(1)`} {
		if got := safeImageURL(bad); got != "" {
			t.Errorf("safeImageURL(%q) = %q, want empty", bad, got)
		}
	}
	for _, ok := range []string{"https://cdn.example.com/logo.png", "/storage/logo.png", "data:image/png;base64,AAAA"} {
		if safeImageURL(ok) == "" {
			t.Errorf("safeImageURL(%q) rejected", ok)
		}
	}
	payload := `<script>fetch('x?'+localStorage.token)</script><img src=x onerror=alert(1)>`
	notes := payload
	html, err := RenderUnifiedDocumentHTML(DocData{Type: "INVOICE", DocumentNoFull: "INV-1", CustomerName: payload, Notes: &notes},
		StoreInfo{Name: payload, LogoURL: "javascript:alert(1)"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<script>fetch") || strings.Contains(html, "<img src=x onerror") || strings.Contains(html, "javascript:alert") {
		t.Fatal("user-controlled fields must be escaped in the preview HTML")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("escaped payload should still be visible as text")
	}
}
