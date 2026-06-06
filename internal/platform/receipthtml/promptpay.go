package receipthtml

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/skip2/go-qrcode"
)

var nonDigitRegex = regexp.MustCompile(`\D`)

// PromptPayQRDataURI builds a base64 PNG data URI for a PromptPay QR code.
// Returns "" when the promptPayID is invalid/empty.
func PromptPayQRDataURI(promptPayID string, amount float64) string {
	payload := promptPayPayload(promptPayID, amount)
	if payload == "" {
		return ""
	}
	png, err := qrcode.Encode(payload, qrcode.Medium, 256)
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

func promptPayPayload(promptPayID string, amount float64) string {
	id := normalizePromptPayID(promptPayID)
	if id == "" {
		return ""
	}
	merchantInfo := ""
	switch len(id) {
	case 13:
		if strings.HasPrefix(id, "0066") {
			merchantInfo = formatEMV("29", formatEMV("00", "A000000677010111")+formatEMV("01", id))
		} else {
			merchantInfo = formatEMV("29", formatEMV("00", "A000000677010111")+formatEMV("02", id))
		}
	case 15:
		merchantInfo = formatEMV("29", formatEMV("00", "A000000677010111")+formatEMV("03", id))
	default:
		return ""
	}
	amountValue := ""
	if amount > 0 {
		amountValue = formatEMV("54", fmt.Sprintf("%.2f", amount))
	}
	raw := "000201" + "010211" + merchantInfo + "5802TH" + "5303764" + amountValue + "6304"
	crc := crc16CCITT(raw)
	return raw + strings.ToUpper(fmt.Sprintf("%04X", crc))
}

func normalizePromptPayID(input string) string {
	digits := nonDigitRegex.ReplaceAllString(strings.TrimSpace(input), "")
	if digits == "" {
		return ""
	}
	if len(digits) == 13 && strings.HasPrefix(digits, "0066") {
		return digits
	}
	if len(digits) == 11 && strings.HasPrefix(digits, "66") {
		return "00" + digits
	}
	if len(digits) == 10 && strings.HasPrefix(digits, "0") {
		return "0066" + digits[1:]
	}
	return digits
}

func formatEMV(tag, value string) string {
	return tag + fmt.Sprintf("%02d", len(value)) + value
}

func crc16CCITT(s string) uint16 {
	const poly uint16 = 0x1021
	var crc uint16 = 0xFFFF
	for i := 0; i < len(s); i++ {
		crc ^= uint16(s[i]) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
