package dochtml

import (
	"fmt"
	"html/template"
	"pos-backend/internal/platform/bizdate"
	"strings"
	"time"
)

var docTitleTH = map[string]string{
	"INVOICE":     "ใบแจ้งหนี้",
	"RECEIPT":     "ใบเสร็จรับเงิน",
	"TAX_INVOICE": "ใบกำกับภาษี",
	"QUOTATION":   "ใบเสนอราคา",
	"BILL":        "ใบวางบิล",
	"CREDIT_NOTE": "ใบลดหนี้",
}

var docTitleEN = map[string]string{
	"INVOICE":     "Invoice",
	"RECEIPT":     "Receipt",
	"TAX_INVOICE": "Tax Invoice",
	"QUOTATION":   "Quotation",
	"BILL":        "Bill",
	"CREDIT_NOTE": "Credit Note",
}

func formatMoney(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	sign := ""
	if strings.HasPrefix(intPart, "-") {
		sign = "-"
		intPart = intPart[1:]
	}
	var result []byte
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return sign + string(result) + "." + parts[1]
}

func fmtThaiDate(t time.Time) string {
	// B02: render the business (Bangkok) calendar day, never the UTC day.
	t = bizdate.Local(t)
	months := [...]string{
		"", "มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
		"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
	}
	return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()], t.Year()+543)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtThaiDate(*t)
}

func isTaxDoc(docType string) bool {
	return docType == "TAX_INVOICE" || docType == "INVOICE"
}

func titleTH(docType string) string { return docTitleTH[docType] }
func titleEN(docType string) string { return docTitleEN[docType] }

// Short aliases used by the unified template builder.
func money(f float64) string      { return formatMoney(f) }
func thaiDate(t time.Time) string { return fmtThaiDate(t) }

// safeImageURL (B29) allows only http(s) URLs and base64 image data URIs for
// values rendered as template.URL (which bypasses html/template's URL
// sanitizer). Anything else — javascript:, data:text/html, quotes — renders
// as no image.
func safeImageURL(raw string) template.URL {
	v := strings.TrimSpace(raw)
	lower := strings.ToLower(v)
	switch {
	case strings.ContainsAny(v, "\"'<>` \t\r\n"):
		return ""
	case strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "http://"), strings.HasPrefix(v, "/"):
		return template.URL(v)
	case strings.HasPrefix(lower, "data:image/png;base64,"), strings.HasPrefix(lower, "data:image/jpeg;base64,"),
		strings.HasPrefix(lower, "data:image/gif;base64,"), strings.HasPrefix(lower, "data:image/webp;base64,"):
		return template.URL(v)
	}
	return ""
}
