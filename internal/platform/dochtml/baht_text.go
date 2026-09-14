package dochtml

import (
	"fmt"
	"math"
	"strings"
)

func formatThaiBahtText(amount float64) string {
	totalSatang := int64(math.Round(amount * 100))
	if totalSatang < 0 {
		totalSatang = 0
	}
	baht, satang := totalSatang/100, totalSatang%100
	text := thaiNumberText(baht) + "บาท"
	if satang == 0 {
		return text + "ถ้วน"
	}
	return text + thaiNumberText(satang) + "สตางค์"
}

func thaiNumberText(n int64) string {
	if n == 0 {
		return "ศูนย์"
	}
	if n >= 1000000 {
		m, r := n/1000000, n%1000000
		text := thaiNumberText(m) + "ล้าน"
		if r > 0 {
			text += thaiNumberText(r)
		}
		return text
	}
	digits := []string{"", "หนึ่ง", "สอง", "สาม", "สี่", "ห้า", "หก", "เจ็ด", "แปด", "เก้า"}
	positions := []string{"", "สิบ", "ร้อย", "พัน", "หมื่น", "แสน"}
	raw := fmt.Sprintf("%d", n)
	var out strings.Builder
	for i, r := range raw {
		digit := int(r - '0')
		if digit == 0 {
			continue
		}
		pos := len(raw) - i - 1
		switch pos {
		case 0:
			if digit == 1 && len(raw) > 1 {
				out.WriteString("เอ็ด")
			} else {
				out.WriteString(digits[digit])
			}
		case 1:
			switch digit {
			case 1:
				out.WriteString("สิบ")
			case 2:
				out.WriteString("ยี่สิบ")
			default:
				out.WriteString(digits[digit] + "สิบ")
			}
		default:
			out.WriteString(digits[digit] + positions[pos])
		}
	}
	return out.String()
}
