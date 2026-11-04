package document

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"gorm.io/gorm"

	"pos-backend/internal/platform/fieldcheck"
)

// Credit notes (B13): always reference an original INVOICE, RECEIPT or
// TAX_INVOICE of the same store, state a reason, and never exceed — together
// with earlier non-cancelled credit notes — the original's total. The check
// runs under a row lock on the original, so concurrent requests cannot both
// pass. Credit notes are accounting documents only; stock returns are
// recorded separately by the sale-return flow.

var creditableTypes = map[DocumentType]bool{TypeInvoice: true, TypeReceipt: true, TypeTaxInvoice: true}

// ErrCreditExceedsOriginal is returned when cumulative credit would exceed
// the original document total.
var ErrCreditExceedsOriginal = errors.New("credit exceeds the original document total")

// CreditExceedsError reports the remaining creditable amount.
type CreditExceedsError struct {
	DocumentNo string
	Remaining  float64
}

func (e *CreditExceedsError) Error() string {
	return fmt.Sprintf("credit exceeds %s: remaining %.2f", e.DocumentNo, e.Remaining)
}

func (e *CreditExceedsError) Unwrap() error { return ErrCreditExceedsOriginal }

type creditReasonKey struct{}

// withCreditReason carries the reason into Convert for CREDIT_NOTE targets.
func withCreditReason(ctx context.Context, reason string) context.Context {
	return context.WithValue(ctx, creditReasonKey{}, reason)
}

func creditReasonFrom(ctx context.Context) string {
	v, _ := ctx.Value(creditReasonKey{}).(string)
	return strings.TrimSpace(v)
}

// checkCreditNoteTx validates a credit note against its locked original and
// stamps the reason. original must already be locked FOR UPDATE in tx.
func checkCreditNoteTx(tx *gorm.DB, original Document, note *Document, reason string) error {
	if err := fieldcheck.Required("reason", reason); err != nil {
		return err
	}
	if !creditableTypes[original.Type] {
		return fieldcheck.New("source_document_id", "credit_source_invalid", "a credit note must reference an invoice, receipt or tax invoice")
	}
	if !CanBeSource(original.Status) {
		return sourceNotEligible(original)
	}
	var credited float64
	if err := tx.Model(&Document{}).
		Where("source_document_id = ? AND type = ? AND status <> ?", original.ID, TypeCreditNote, StatusCancelled).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&credited).Error; err != nil {
		return err
	}
	remaining := math.Round((original.TotalAmount-credited)*100) / 100
	if note.TotalAmount <= 0 || math.IsNaN(note.TotalAmount) {
		return fieldcheck.New("total_amount", fieldcheck.CodeNegative, "credit amount must be greater than zero")
	}
	if note.TotalAmount > remaining+0.005 {
		return &CreditExceedsError{DocumentNo: original.DocumentNoFull, Remaining: math.Max(remaining, 0)}
	}
	note.CreditReason = &reason
	return nil
}
