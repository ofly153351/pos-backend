package sale

import "testing"

func TestFormatThaiBahtText(t *testing.T) {
	tests := map[float64]string{
		0:       "ศูนย์บาทถ้วน",
		11:      "สิบเอ็ดบาทถ้วน",
		21:      "ยี่สิบเอ็ดบาทถ้วน",
		101:     "หนึ่งร้อยเอ็ดบาทถ้วน",
		1300:    "หนึ่งพันสามร้อยบาทถ้วน",
		1300.50: "หนึ่งพันสามร้อยบาทห้าสิบสตางค์",
	}

	for input, want := range tests {
		if got := formatThaiBahtText(input); got != want {
			t.Fatalf("formatThaiBahtText(%v) = %q, want %q", input, got, want)
		}
	}
}
