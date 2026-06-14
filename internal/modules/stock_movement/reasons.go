package stock_movement

import "strings"

// Adjustment operation kinds used for reason-code validation. They map to the three
// user-facing quick-adjustment modes; each endpoint validates the submitted reason
// against its kind's allowed code set (Phase W2 §13).
const (
	opAdd       = "ADD"        // เพิ่มสต็อก  → IN movement
	opSubtract  = "SUBTRACT"   // ลดสต็อก    → OUT movement
	opSetActual = "SET_ACTUAL" // กำหนดยอดจริง → ADJUST movement
)

// reasonCodes is the stable, language-independent set of valid reason codes per
// operation. The frontend renders localized labels; the backend stores and validates
// the CODE so reporting stays language-agnostic.
var reasonCodes = map[string]map[string]struct{}{
	opAdd: {
		"FOUND_EXTRA":     {},
		"RETURN_TO_STOCK": {},
		"OPENING_BALANCE": {},
		"DATA_CORRECTION": {},
		"OTHER":           {},
	},
	opSubtract: {
		"DAMAGED":         {},
		"LOST":            {},
		"EXPIRED":         {},
		"INTERNAL_USE":    {},
		"WRITE_OFF":       {},
		"DATA_CORRECTION": {},
		"OTHER":           {},
	},
	opSetActual: {
		"SPOT_COUNT":      {},
		"SYSTEM_MISMATCH": {},
		"DATA_CORRECTION": {},
		"OTHER":           {},
	},
}

// validateReason enforces that a reason is supplied, is valid for the operation, and
// that the free-text note is present when the catch-all OTHER code is used.
func validateReason(op, reason, note string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrStockReasonRequired
	}
	set, ok := reasonCodes[op]
	if !ok {
		return ErrStockReasonInvalid
	}
	if _, valid := set[reason]; !valid {
		return ErrStockReasonInvalid
	}
	if reason == "OTHER" && strings.TrimSpace(note) == "" {
		return ErrStockReasonNoteRequired
	}
	return nil
}
