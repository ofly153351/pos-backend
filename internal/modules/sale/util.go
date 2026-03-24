package sale

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

const saleStatusCompleted = "completed"
const (
	DiscountTypeAmount  = "amount"
	DiscountTypePercent = "percent"
)

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "generated-id"
	}
	return hex.EncodeToString(buf)
}

func newSaleNumber(now time.Time) string {
	return fmt.Sprintf("SALE-%s-%s", now.UTC().Format("20060102150405"), newID()[:6])
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

func normalizeDiscountType(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
