package fieldcheck

import (
	"fmt"
	"math"
	"testing"
)

func code(err error) string {
	if fe, ok := As(err); ok {
		return fe.Code
	}
	return ""
}

func TestTaxID(t *testing.T) {
	valid := []string{"", "1234567890121", "0105536000313", "0-1055-36000-31-3", " 0105536000313 ", "3100600445597"}
	for _, v := range valid {
		if err := TaxID("tax_id", v); err != nil {
			t.Errorf("TaxID(%q) = %v, want valid", v, err)
		}
	}
	invalid := []string{"123", "1234567890123", "0105536000315", "12345678901a1", "12345678901234", "๑๒๓๔๕๖๗๘๙๐๑๒๑"}
	for _, v := range invalid {
		if err := TaxID("tax_id", v); code(err) != CodeTaxIDInvalid {
			t.Errorf("TaxID(%q) = %v, want %s", v, err, CodeTaxIDInvalid)
		}
	}
	if NormalizeTaxID(" 0-1055-36000-31-3 ") != "0105536000313" {
		t.Fatal("NormalizeTaxID did not strip separators")
	}
}

func TestPhone(t *testing.T) {
	for _, v := range []string{"", "021234567", "081-234-5678", "(02) 123 4567", "+66 81 234 5678", "02.123.4567"} {
		if err := Phone("phone", v); err != nil {
			t.Errorf("Phone(%q) = %v", v, err)
		}
	}
	for _, v := range []string{"12345", "abc0812345678", "081+2345678", "1234567890123456", "๐๘๑๒๓๔๕๖๗๘"} {
		if err := Phone("phone", v); code(err) != CodePhoneInvalid {
			t.Errorf("Phone(%q) = %v, want %s", v, err, CodePhoneInvalid)
		}
	}
}

func TestNumbers(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{Percent("p", 0), ""}, {Percent("p", 100), ""}, {Percent("p", 12.5), ""},
		{Percent("p", -0.01), CodePercentRange}, {Percent("p", 150), CodePercentRange},
		{Percent("p", math.NaN()), CodeNotFinite}, {NonNegative("n", math.Inf(1)), CodeNotFinite},
		{NonNegative("n", 0), ""}, {NonNegative("n", -1), CodeNegative},
		{Required("r", "  "), CodeRequired}, {Required("r", "x"), ""},
	}
	for i, c := range cases {
		if got := code(c.err); got != c.want {
			t.Errorf("case %d: code %q want %q (%v)", i, got, c.want, c.err)
		}
	}
}

func TestAsThroughWrap(t *testing.T) {
	err := fmt.Errorf("update: %w", TaxID("tax_id", "123"))
	fe, ok := As(err)
	if !ok || fe.Field != "tax_id" || fe.Code != CodeTaxIDInvalid {
		t.Fatalf("As = %+v %v", fe, ok)
	}
	if First(nil, nil, Required("a", "")) == nil || First(nil) != nil {
		t.Fatal("First")
	}
}
