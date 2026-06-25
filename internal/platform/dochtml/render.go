package dochtml

import (
	"bytes"
	"fmt"
)

// RenderDocumentHTML renders any document type via the unified template.
func RenderDocumentHTML(doc DocData, store StoreInfo) (string, error) {
	return RenderUnifiedDocumentHTML(doc, store)
}

// RenderWHTCertHTML renders the WHT certificate (ภ.ง.ด.3/53) HTML.
func RenderWHTCertHTML(d WHTCertData) (string, error) {
	var buf bytes.Buffer
	if err := whtTmpl.Execute(&buf, d); err != nil {
		return "", fmt.Errorf("wht html: %w", err)
	}
	return buf.String(), nil
}
