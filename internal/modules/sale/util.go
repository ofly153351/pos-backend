package sale

import (
	"fmt"
	"math"
	"pos-backend/internal/idgen"
	"strings"
	"time"
)

const saleStatusCompleted = "completed"

// SaleStatusVoided marks a sale that has been reversed (e.g. a cancelled credit
// sale whose goods were restocked). Voided sales are excluded from all
// revenue/COGS reporting.
const SaleStatusVoided = "voided"

// Partial-return statuses (migration 052). The sale row stays; only some-or-all
// item quantities are returned. Distinct from voided (entire-bill cancel).
const (
	SaleStatusPartiallyReturned = "partially_returned"
	SaleStatusFullyReturned     = "fully_returned"
)

func newSaleReturnID() string     { return idgen.Generate(idgen.PrefixSaleReturn) }
func newSaleReturnItemID() string { return idgen.Generate(idgen.PrefixSaleReturnItem) }

func newReturnNumber(now time.Time) string {
	id := idgen.Generate(idgen.PrefixSaleReturn)
	return fmt.Sprintf("RET-%s-%s", now.UTC().Format("20060102150405"), id[len(id)-6:])
}
const (
	DiscountTypeAmount  = "amount"
	DiscountTypePercent = "percent"
)

func newID() string { return idgen.Generate(idgen.PrefixSale) }

func newSaleItemID() string { return idgen.Generate(idgen.PrefixSaleItem) }

func newStockMovementID() string { return idgen.Generate(idgen.PrefixStockMovement) }

func newPromotionUsageID() string { return idgen.Generate(idgen.PrefixPromotionUsage) }

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
