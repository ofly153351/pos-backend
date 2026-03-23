package invoice

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	discountTypeAmount  = "amount"
	discountTypePercent = "percent"
)

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "generated-id"
	}
	return hex.EncodeToString(buf)
}

func newInvoiceNumber(now time.Time) string {
	return fmt.Sprintf("INV-%s-%s", now.UTC().Format("20060102150405"), newID()[:6])
}

func normalizeDiscountType(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func resolveEffectivePrice(product productSnapshot, now time.Time) float64 {
	if product.SpecialPrice == nil {
		return product.BasePrice
	}
	if product.SpecialPriceStartAt != nil && now.Before(*product.SpecialPriceStartAt) {
		return product.BasePrice
	}
	if product.SpecialPriceEndAt != nil && now.After(*product.SpecialPriceEndAt) {
		return product.BasePrice
	}
	return *product.SpecialPrice
}

func calculateDiscount(discountType string, discountValue *float64, unitPrice float64) (float64, error) {
	normalizedType := normalizeDiscountType(discountType)
	if normalizedType == "" {
		if discountValue != nil {
			return 0, ErrInvalidDiscountType
		}
		return 0, nil
	}
	if discountValue == nil {
		return 0, ErrDiscountValueRequired
	}
	if *discountValue < 0 {
		return 0, ErrInvalidDiscountValue
	}

	switch normalizedType {
	case discountTypeAmount:
		if *discountValue > unitPrice {
			return 0, ErrAmountDiscountExceeds
		}
		return *discountValue, nil
	case discountTypePercent:
		if *discountValue > 100 {
			return 0, ErrInvalidPercentDiscount
		}
		return unitPrice * (*discountValue / 100), nil
	default:
		return 0, ErrInvalidDiscountType
	}
}

func calculateNetworkDiscount(percent, unitPrice, manualDiscount float64) float64 {
	if percent <= 0 {
		return 0
	}
	base := unitPrice - manualDiscount
	if base <= 0 {
		return 0
	}
	return base * (percent / 100)
}
