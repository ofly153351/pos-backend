package receipt_settings

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Embed only the selected store's own logo for the iframe. Its persisted URL
// may point at an internal asset host not reachable by the user's browser.
// Keep this local to settings preview; real print rendering is unchanged.
func previewStoreLogo(ctx context.Context, raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "data:image/") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, raw, nil)
	if err != nil {
		return raw
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return raw
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return raw
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, (5<<20)+1))
	if err != nil || len(content) == 0 || len(content) > 5<<20 {
		return raw
	}
	kind := http.DetectContentType(content)
	if !strings.HasPrefix(kind, "image/") {
		return raw
	}
	return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(content)
}
