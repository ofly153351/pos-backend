package sale

import (
	"fmt"
	"math"
	"pos-backend/internal/idgen"
	"strings"
	"time"
)

const saleStatusCompleted = "completed"
const (
	DiscountTypeAmount  = "amount"
	DiscountTypePercent = "percent"
)

func newID() string { return idgen.Generate(idgen.PrefixSale) }

func newSaleItemID() string { return idgen.Generate(idgen.PrefixSaleItem) }

func newStockMovementID() string { return idgen.Generate(idgen.PrefixStockMovement) }

func newSaleNumber(now time.Time) string {
	id := idgen.Generate(idgen.PrefixSale)
	return fmt.Sprintf("SALE-%s-%s", now.UTC().Format("20060102150405"), id[len(id)-6:])
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
