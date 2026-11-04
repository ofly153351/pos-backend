// Package fieldcheck holds shared backend input rules (B10/B16) and the typed
// field error that carries a stable machine code for localized messages (B23).
//
// Rules return *Error (or nil). Handlers turn it into a 422 with
// {field, code, message} via httpx.FieldErrorResponse, so the frontend can
// translate `code` through th.json/en.json and still show `message` for codes
// it does not know.
package fieldcheck

import (
	"errors"
	"math"
	"strings"
	"unicode"
)

// Stable error codes. Never rename: the frontend dictionaries key on them.
const (
	CodeRequired     = "required"
	CodeTaxIDInvalid = "tax_id_invalid"
	CodePhoneInvalid = "phone_invalid"
	CodePercentRange = "percent_out_of_range"
	CodeNegative     = "must_not_be_negative"
	CodeNotFinite    = "must_be_number"
	CodeGreaterThan  = "must_not_exceed"
)

// Error is a single invalid field.
type Error struct {
	Field   string
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Field + ": " + e.Message }

// As extracts a field error from err.
func As(err error) (*Error, bool) {
	var fe *Error
	if errors.As(err, &fe) {
		return fe, true
	}
	return nil, false
}

func fail(field, code, message string) error {
	return &Error{Field: field, Code: code, Message: message}
}

// NormalizeTaxID removes the separators people commonly type (spaces, dashes).
func NormalizeTaxID(value string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(value) {
		if r == ' ' || r == '-' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// TaxID validates a Thai taxpayer identification number: 13 digits with the
// mod-11 check digit. The same rule applies to individual citizen IDs and
// juristic-person numbers. Empty means "not provided" and is allowed; callers
// that require a tax ID check emptiness separately.
func TaxID(field, value string) error {
	v := NormalizeTaxID(value)
	if v == "" {
		return nil
	}
	if len(v) != 13 {
		return fail(field, CodeTaxIDInvalid, "tax ID must be 13 digits")
	}
	sum := 0
	for i := 0; i < 12; i++ {
		c := v[i]
		if c < '0' || c > '9' {
			return fail(field, CodeTaxIDInvalid, "tax ID must be 13 digits")
		}
		sum += int(c-'0') * (13 - i)
	}
	last := v[12]
	if last < '0' || last > '9' {
		return fail(field, CodeTaxIDInvalid, "tax ID must be 13 digits")
	}
	if (11-sum%11)%10 != int(last-'0') {
		return fail(field, CodeTaxIDInvalid, "tax ID check digit is invalid")
	}
	return nil
}

// Phone accepts digits with common separators and an optional leading +.
// It requires 9-15 digits (Thai landline 9, mobile 10, international up to 15).
// Empty is allowed.
func Phone(field, value string) error {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	digits := 0
	for i, r := range v {
		switch {
		case unicode.IsDigit(r) && r < 128:
			digits++
		case r == ' ' || r == '-' || r == '(' || r == ')' || r == '.':
		case r == '+' && i == 0:
		default:
			return fail(field, CodePhoneInvalid, "phone number may contain only digits, spaces, dashes, parentheses and a leading +")
		}
	}
	if digits < 9 || digits > 15 {
		return fail(field, CodePhoneInvalid, "phone number must have 9-15 digits")
	}
	return nil
}

// Percent requires a finite value between 0 and 100 inclusive.
func Percent(field string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fail(field, CodeNotFinite, "must be a number")
	}
	if v < 0 || v > 100 {
		return fail(field, CodePercentRange, "must be between 0 and 100")
	}
	return nil
}

// NonNegative requires a finite value >= 0 (prices, costs, quantities).
func NonNegative(field string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fail(field, CodeNotFinite, "must be a number")
	}
	if v < 0 {
		return fail(field, CodeNegative, "must not be negative")
	}
	return nil
}

// Required rejects empty or whitespace-only values.
func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fail(field, CodeRequired, "is required")
	}
	return nil
}

// New builds a field error with an explicit code, for module-specific rules.
func New(field, code, message string) error { return fail(field, code, message) }

// First returns the first non-nil error.
func First(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Rule is a string-field validator such as TaxID or Phone.
type Rule func(field, value string) error

// IfChanged validates a submitted value only when it differs from the stored
// one, so pre-existing invalid data never blocks an unrelated edit (owner rule:
// legacy values fail only when that field is edited). nil means "not sent".
func IfChanged(field string, submitted *string, stored string, rule Rule) error {
	if submitted == nil || *submitted == stored {
		return nil
	}
	return rule(field, *submitted)
}
