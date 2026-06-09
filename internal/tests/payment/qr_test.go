package payment_test

import (
	"strings"
	"testing"

	"pos-backend/internal/platform/receipthtml"
)

// The endpoint delegates QR generation to receipthtml.PromptPayQRDataURI;
// verify the contract the handler relies on.
func TestPromptPayQR_AmountMatches(t *testing.T) {
	a := receipthtml.PromptPayQRDataURI("0812345678", 100.00)
	b := receipthtml.PromptPayQRDataURI("0812345678", 268.00)
	if a == "" || b == "" {
		t.Fatal("expected non-empty QR for valid id")
	}
	if a == b {
		t.Error("QR must differ when amount differs (dynamic QR)")
	}
	if !strings.HasPrefix(a, "data:image/png;base64,") {
		t.Errorf("expected png data URI, got %.30q", a)
	}
}

func TestPromptPayQR_InvalidEmpty(t *testing.T) {
	if receipthtml.PromptPayQRDataURI("", 100) != "" {
		t.Error("empty id must yield empty QR (handler returns 422)")
	}
}
