package dochtml

import (
	"fmt"
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
