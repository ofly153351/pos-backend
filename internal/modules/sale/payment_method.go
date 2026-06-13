package sale

import "strings"

// AllowedPaymentMethods mirrors the expense module's canonical vocabulary so both
// modules validate payment methods against the same set. The six canonical keys
// are accepted, plus the legacy keys still produced by the current checkout UI
// ("transfer" is the PromptPay/QR flow), kept valid so existing rows and the live
// cashier keep working until a UI migration consolidates the vocabulary.
var AllowedPaymentMethods = map[string]bool{
	"cash":          true,
	"bank_transfer": true,
	"promptpay":     true,
	"credit_card":   true,
	"debit_card":    true,
	"cheque":        true,
	// legacy / current-UI keys — accepted, not canonical
	"transfer": true,
	"qr":       true,
	"credit":   true,
	"card":     true,
}

// normalizePaymentMethod trims and lower-cases the raw value so casing/whitespace
// variants collapse to one key. It deliberately does NOT remap legacy keys to a
// canonical one (e.g. transfer→promptpay): that would change the meaning of
// already-stored sales and the labels rendered from them. Remapping is a separate,
// deliberate data migration.
func normalizePaymentMethod(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
