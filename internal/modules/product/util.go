package product

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "generated-id"
	}

	return hex.EncodeToString(buf)
}

func newBarcode12Digits() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "000000000000"
	}

	var out strings.Builder
	out.Grow(12)
	for _, v := range b {
		out.WriteByte('0' + (v % 10))
	}
	return out.String()
}

func ean13CheckDigit(d12 string) (int, error) {
	if len(d12) != 12 {
		return 0, fmt.Errorf("invalid ean12 length")
	}
	sum := 0
	for i := 0; i < 12; i++ {
		ch := d12[i]
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("ean12 must contain digits only")
		}
		d := int(ch - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return (10 - (sum % 10)) % 10, nil
}

func buildEAN13(d12 string) string {
	check, err := ean13CheckDigit(d12)
	if err != nil {
		return d12 + "0"
	}
	return fmt.Sprintf("%s%d", d12, check)
}
