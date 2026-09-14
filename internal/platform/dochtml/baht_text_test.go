package dochtml

import "testing"

func TestFormatThaiBahtText(t *testing.T) {
	cases := map[float64]string{
		238.61: "สองร้อยสามสิบแปดบาทหกสิบเอ็ดสตางค์",
		200:    "สองร้อยบาทถ้วน",
		0:      "ศูนย์บาทถ้วน",
	}
	for amount, want := range cases {
		if got := formatThaiBahtText(amount); got != want {
			t.Errorf("formatThaiBahtText(%v) = %q, want %q", amount, got, want)
		}
	}
}
