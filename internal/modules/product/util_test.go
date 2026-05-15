package product

import "testing"

func TestBuildEAN13(t *testing.T) {
	got := buildEAN13("885012345678")
	want := "8850123456787"
	if got != want {
		t.Fatalf("buildEAN13() = %q, want %q", got, want)
	}
}

func TestNewBarcode12Digits(t *testing.T) {
	code := newBarcode12Digits()
	if len(code) != 12 {
		t.Fatalf("newBarcode12Digits length = %d, want 12", len(code))
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			t.Fatalf("newBarcode12Digits contains non-digit: %q", code)
		}
	}
}
