package dochtml

import (
	"bytes"
	"fmt"
	"html/template"
)

const (
	typeBill           = "BILL"
	itemsPerFirstPage  = 25 // allows for logo + full header
	itemsPerOtherPage  = 35 // compact header on continuation pages
)

var funcMap = template.FuncMap{
	"money":     formatMoney,
	"thaiDate":  fmtThaiDate,
	"derefStr":  derefStr,
	"derefTime": derefTime,
	"titleTH":   titleTH,
	"titleEN":   titleEN,
	"isTaxDoc":  isTaxDoc,
	"inc":       func(i int) int { return i + 1 },
	"add":       func(a, b int) int { return a + b },
	"fmtQty":    func(f float64) string { return fmt.Sprintf("%.0f", f) },
}

// RenderDocumentHTML selects the appropriate HTML template by document type.
func RenderDocumentHTML(doc DocData, store StoreInfo) (string, error) {
	if doc.Type == typeBill {
		return renderBillHTML(doc, store)
	}
	pages := paginateInvoice(doc, store)
	var buf bytes.Buffer
	if err := invoiceTmpl.Execute(&buf, renderData{Pages: pages}); err != nil {
		return "", fmt.Errorf("invoice html: %w", err)
	}
	return buf.String(), nil
}

// paginateInvoice splits items into pages.
func paginateInvoice(doc DocData, store StoreInfo) []pageData {
	items := doc.Items

	// Collect page groups
	var groups [][]DocItem
	if len(items) <= itemsPerFirstPage {
		groups = append(groups, items)
	} else {
		groups = append(groups, items[:itemsPerFirstPage])
		rest := items[itemsPerFirstPage:]
		for len(rest) > 0 {
			end := itemsPerOtherPage
			if end > len(rest) {
				end = len(rest)
			}
			groups = append(groups, rest[:end])
			rest = rest[end:]
		}
	}

	total := len(groups)
	pages := make([]pageData, total)
	for i, group := range groups {
		isLast := i == total-1
		fillerCount := 0
		if isLast {
			limit := itemsPerFirstPage
			if i > 0 {
				limit = itemsPerOtherPage
			}
			fc := limit - len(group)
			if fc > 0 && fc <= 5 {
				fillerCount = fc
			}
		}
		offset := 0
		if i > 0 {
			offset = itemsPerFirstPage + (i-1)*itemsPerOtherPage
		}
		pages[i] = pageData{
			Doc:        doc,
			Store:      store,
			Items:      group,
			FillerRows: make([]struct{}, fillerCount),
			ItemOffset: offset,
			PageNo:     i + 1,
			TotalPages: total,
			IsFirst:    i == 0,
			IsLast:     isLast,
		}
	}
	return pages
}

func renderBillHTML(doc DocData, store StoreInfo) (string, error) {
	var buf bytes.Buffer
	if err := billTmpl.Execute(&buf, billRenderData{Doc: doc, Store: store}); err != nil {
		return "", fmt.Errorf("bill html: %w", err)
	}
	return buf.String(), nil
}

// RenderWHTCertHTML renders the WHT certificate (ภ.ง.ด.3/53) HTML.
func RenderWHTCertHTML(d WHTCertData) (string, error) {
	var buf bytes.Buffer
	if err := whtTmpl.Execute(&buf, d); err != nil {
		return "", fmt.Errorf("wht html: %w", err)
	}
	return buf.String(), nil
}
