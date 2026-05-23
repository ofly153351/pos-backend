package invoice

import (
	"fmt"
	"math"
	"pos-backend/internal/idgen"
	"strings"
	"time"
)

const (
	discountTypeAmount  = "amount"
	discountTypePercent = "percent"
)

func newID() string { return idgen.Generate(idgen.PrefixInvoice) }

func newInvoiceItemID() string { return idgen.Generate(idgen.PrefixInvoiceItem) }

func newInvoicePaymentID() string { return idgen.Generate(idgen.PrefixInvoicePayment) }

func newStockMovementID() string { return idgen.Generate(idgen.PrefixStockMovement) }

func newFileToken() string { return fmt.Sprintf("%08d", idgen.NextInt()) }

func newInvoiceNumber(now time.Time) string {
	id := idgen.Generate(idgen.PrefixInvoice)
	return fmt.Sprintf("INV-%s-%s", now.UTC().Format("20060102150405"), id[len(id)-6:])
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

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
