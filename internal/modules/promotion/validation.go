package promotion

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// ValidateDiscountParameters guards both campaign writes and legacy campaigns at
// checkout. Optional scheduling/scope metadata is deliberately not required.
func ValidateDiscountParameters(promoType string, data []byte) error {
	return validateDiscountParameters(promoType, data, "active")
}

// Drafts may omit unfinished parameters, but supplied values must be valid.
// Pausing must remain possible for invalid historical campaigns. Checkout always
// uses strict validation and never grants an incomplete/invalid campaign a discount.
func validateDiscountParameters(promoType string, data []byte, status string) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return ErrInvalidBody
	}
	// Go struct decoding matches JSON keys case-insensitively, whereas maps do
	// not. Reject aliases so metadata and checkout cannot read different values.
	canonicalKeys := []string{"id", "name", "type", "status", "code", "percentOff", "amountOff", "discountAmount", "exchangeDiscount", "bundlePrice", "fixedPrice", "memberPrice", "buyQty", "getQty", "startDate", "endDate"}
	for key := range obj {
		for _, canonical := range canonicalKeys {
			if key != canonical && strings.EqualFold(key, canonical) {
				return fmt.Errorf("%w: non-canonical %s", ErrInvalidBody, key)
			}
		}
	}
	if status == "paused" {
		return nil
	}
	var fields []string
	switch promoType {
	case "percentage", "happy_hour":
		fields = []string{"percentOff"}
	case "fixed_amount", "coupon":
		fields = []string{"amountOff"}
	case "spend_x_discount":
		fields = []string{"discountAmount"}
	case "cylinder_exchange":
		fields = []string{"exchangeDiscount"}
	case "bundle":
		fields = []string{"bundlePrice"}
	case "fixed_price":
		fields = []string{"fixedPrice"}
	case "member_price":
		fields = []string{"memberPrice"}
	case "buy_x_get_y":
		fields = []string{"buyQty", "getQty"}
	}
	for _, field := range fields {
		raw, ok := obj[field]
		if !ok && status == "draft" {
			continue
		}
		var value float64
		if !ok || string(raw) == "null" || json.Unmarshal(raw, &value) != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || (field == "percentOff" && value > 100) || ((field == "buyQty" || field == "getQty") && value <= 0) {
			return fmt.Errorf("%w: invalid %s", ErrInvalidBody, field)
		}
	}
	return nil
}
