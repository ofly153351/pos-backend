package document

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pos-backend/internal/idgen"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/bizdate"
	"pos-backend/internal/platform/fieldcheck"
)

// Document payments (B05) live in the existing invoice_payments ledger
// (owner decision 2026-11-04), keyed by document_id.
//
// Debt rule (owner, 2026-11-04): a debt is owned by its payable root — the
// INVOICE a document descends from, or the document itself when it has no
// invoice ancestor. Delivery orders, tax invoices and bills therefore never
// add debt of their own when they derive from an invoice; a BILL with linked
// delivery orders settles their roots. Remaining = total − payments − credit
// notes. Partial payments are allowed; overpayment is rejected.

const PrefixPayment = "dpay"

// DocumentPayment is a row of invoice_payments that targets a document.
type DocumentPayment struct {
	ID             string    `gorm:"column:id;primaryKey" json:"id"`
	DocumentID     *string   `gorm:"column:document_id" json:"document_id"`
	StoreID        string    `gorm:"column:store_id" json:"store_id"`
	Amount         float64   `gorm:"column:paid_amount" json:"amount"`
	Method         string    `gorm:"column:payment_method" json:"method"`
	Note           *string   `gorm:"column:note" json:"note,omitempty"`
	PaidAt         time.Time `gorm:"column:paid_at" json:"paid_at"`
	CreatedBy      string    `gorm:"column:created_by" json:"created_by"`
	IdempotencyKey *string   `gorm:"column:idempotency_key" json:"-"`
	IsVoided       bool      `gorm:"column:is_voided" json:"is_voided"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (DocumentPayment) TableName() string { return "invoice_payments" }

// DocumentBalance is the debt of one payable root.
type DocumentBalance struct {
	DocumentID    string        `json:"document_id"`
	DocumentNo    string        `json:"document_no"`
	Total         float64       `json:"total"`
	Paid          float64       `json:"paid"`
	Credited      float64       `json:"credited"`
	Remaining     float64       `json:"remaining"`
	PaymentStatus PaymentStatus `json:"payment_status"`
}

// PaymentSummary answers balance and payment requests for a document.
type PaymentSummary struct {
	Balances  []DocumentBalance `json:"balances"`
	Total     float64           `json:"total"`
	Paid      float64           `json:"paid"`
	Credited  float64           `json:"credited"`
	Remaining float64           `json:"remaining"`
	Payments  []DocumentPayment `json:"payments"`
	Replayed  bool              `json:"replayed,omitempty"`
}

type RecordPaymentRequest struct {
	Amount   float64 `json:"amount"`
	Method   string  `json:"method"`
	PaidDate string  `json:"paid_date"` // YYYY-MM-DD business date; default today
	Note     string  `json:"note"`
}

var paymentMethods = map[string]bool{"cash": true, "transfer": true, "qr": true, "card": true, "cheque": true}

// ErrOverpayment is returned when a payment exceeds the remaining debt.
var ErrOverpayment = errors.New("payment exceeds the remaining balance")

type OverpaymentError struct{ Remaining float64 }

func (e *OverpaymentError) Error() string {
	return fmt.Sprintf("payment exceeds the remaining balance %.2f", e.Remaining)
}
func (e *OverpaymentError) Unwrap() error { return ErrOverpayment }

func money(v float64) float64 { return math.Round(v*100) / 100 }

// payableRootsTx returns the locked payable roots for doc (see package rule).
func payableRootsTx(tx *gorm.DB, storeID string, doc Document) ([]Document, error) {
	var starts []Document
	if doc.Type == TypeBill {
		var linked []Document
		if err := tx.Where("store_id = ? AND id IN (SELECT delivery_order_id FROM bill_delivery_order_links WHERE bill_document_id = ?)", storeID, doc.ID).
			Order("document_date, id").Find(&linked).Error; err != nil {
			return nil, err
		}
		starts = linked
	}
	if len(starts) == 0 {
		starts = []Document{doc}
	}
	seen := map[string]bool{}
	var roots []Document
	for _, start := range starts {
		root := start
		for depth := 0; root.Type != TypeInvoice && root.SourceDocumentID != nil && *root.SourceDocumentID != "" && depth < 8; depth++ {
			var parent Document
			if err := tx.First(&parent, "id = ? AND store_id = ?", *root.SourceDocumentID, storeID).Error; err != nil {
				break
			}
			if parent.Type == TypeQuotation {
				break // a quotation is not a debt
			}
			root = parent
		}
		if seen[root.ID] {
			continue
		}
		seen[root.ID] = true
		roots = append(roots, root)
	}
	ids := make([]string, 0, len(roots))
	for _, r := range roots {
		ids = append(ids, r.ID)
	}
	// Lock every root so concurrent payments serialize on the same debt.
	var locked []Document
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Order("id").Find(&locked).Error; err != nil {
		return nil, err
	}
	byID := map[string]Document{}
	for _, d := range locked {
		byID[d.ID] = d
	}
	out := make([]Document, 0, len(roots))
	for _, r := range roots {
		out = append(out, byID[r.ID])
	}
	return out, nil
}

// balanceTx computes the debt of one root: credit notes count against the
// root's whole family (the root and every document derived from it).
func balanceTx(tx *gorm.DB, root Document) (DocumentBalance, error) {
	var paid, credited float64
	if err := tx.Model(&DocumentPayment{}).Where("document_id = ? AND is_voided = false", root.ID).
		Select("COALESCE(SUM(paid_amount), 0)").Scan(&paid).Error; err != nil {
		return DocumentBalance{}, err
	}
	if err := tx.Raw(`WITH RECURSIVE fam AS (
			SELECT id FROM documents WHERE id = ?
			UNION SELECT d.id FROM documents d JOIN fam ON d.source_document_id = fam.id
		)
		SELECT COALESCE(SUM(total_amount), 0) FROM documents
		WHERE type = 'CREDIT_NOTE' AND status <> 'CANCELLED' AND source_document_id IN (SELECT id FROM fam)`, root.ID).
		Scan(&credited).Error; err != nil {
		return DocumentBalance{}, err
	}
	b := DocumentBalance{
		DocumentID: root.ID, DocumentNo: root.DocumentNoFull,
		Total: money(root.TotalAmount), Paid: money(paid), Credited: money(credited),
	}
	b.Remaining = money(math.Max(b.Total-b.Paid-b.Credited, 0))
	switch {
	case b.Remaining <= 0.005:
		b.PaymentStatus = PaymentPaid
	case b.Paid > 0 || b.Credited > 0:
		b.PaymentStatus = PaymentPartial
	default:
		b.PaymentStatus = PaymentUnpaid
	}
	return b, nil
}

func summarize(balances []DocumentBalance, payments []DocumentPayment) PaymentSummary {
	s := PaymentSummary{Balances: balances, Payments: payments}
	for _, b := range balances {
		s.Total += b.Total
		s.Paid += b.Paid
		s.Credited += b.Credited
		s.Remaining += b.Remaining
	}
	s.Total, s.Paid, s.Credited, s.Remaining = money(s.Total), money(s.Paid), money(s.Credited), money(s.Remaining)
	if s.Payments == nil {
		s.Payments = []DocumentPayment{}
	}
	return s
}

func payableTarget(doc Document) error {
	switch doc.Type {
	case TypeQuotation, TypeCreditNote:
		return fieldcheck.New("document_id", "not_payable", "this document type does not carry a debt")
	}
	if !CanBeSource(doc.Status) {
		return sourceNotEligible(doc)
	}
	return nil
}

// PaymentSummary returns balances and payments for the debt a document carries.
func (s Service) PaymentSummary(ctx context.Context, actor auth.Claims, storeID, id string) (PaymentSummary, error) {
	var out PaymentSummary
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		doc, err := lockDocumentTx(tx, storeID, id)
		if err != nil {
			return err
		}
		roots, err := payableRootsTx(tx, storeID, doc)
		if err != nil {
			return err
		}
		out, err = summaryTx(tx, roots)
		return err
	})
	return out, err
}

func summaryTx(tx *gorm.DB, roots []Document) (PaymentSummary, error) {
	balances := make([]DocumentBalance, 0, len(roots))
	ids := make([]string, 0, len(roots))
	for _, r := range roots {
		b, err := balanceTx(tx, r)
		if err != nil {
			return PaymentSummary{}, err
		}
		balances = append(balances, b)
		ids = append(ids, r.ID)
	}
	var payments []DocumentPayment
	if err := tx.Where("document_id IN ? AND is_voided = false", ids).Order("paid_at, created_at").Find(&payments).Error; err != nil {
		return PaymentSummary{}, err
	}
	return summarize(balances, payments), nil
}

// RecordPayment records a (partial) payment against a document's debt in one
// transaction: roots locked, overpayment rejected, duplicate keys replayed,
// and payment/lifecycle status of the whole family updated.
func (s Service) RecordPayment(ctx context.Context, actor auth.Claims, storeID, id string, req RecordPaymentRequest, idempotencyKey string) (PaymentSummary, error) {
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if err := fieldcheck.NonNegative("amount", req.Amount); err != nil {
		return PaymentSummary{}, err
	}
	if req.Amount <= 0 {
		return PaymentSummary{}, fieldcheck.New("amount", fieldcheck.CodeNegative, "amount must be greater than zero")
	}
	if !paymentMethods[method] {
		return PaymentSummary{}, fieldcheck.New("method", "payment_method_invalid", "payment method must be cash, transfer, qr, card or cheque")
	}
	paidAt := bizdate.Today()
	if strings.TrimSpace(req.PaidDate) != "" {
		d, err := bizdate.Parse(strings.TrimSpace(req.PaidDate))
		if err != nil {
			return PaymentSummary{}, fieldcheck.New("paid_date", "date_invalid", "paid_date must be YYYY-MM-DD")
		}
		paidAt = d
	}
	key := strings.TrimSpace(idempotencyKey)
	amount := money(req.Amount)

	var out PaymentSummary
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		doc, err := lockDocumentTx(tx, storeID, id)
		if err != nil {
			return err
		}
		if err := payableTarget(doc); err != nil {
			return err
		}
		roots, err := payableRootsTx(tx, storeID, doc)
		if err != nil {
			return err
		}
		rootIDs := make([]string, 0, len(roots))
		for _, r := range roots {
			if !CanBeSource(r.Status) {
				return sourceNotEligible(r)
			}
			rootIDs = append(rootIDs, r.ID)
		}
		if key != "" {
			var replay int64
			if err := tx.Model(&DocumentPayment{}).Where("document_id IN ? AND idempotency_key = ?", rootIDs, key).Count(&replay).Error; err != nil {
				return err
			}
			if replay > 0 {
				out, err = summaryTx(tx, roots)
				out.Replayed = true
				return err
			}
		}
		balances := make([]DocumentBalance, 0, len(roots))
		remaining := 0.0
		for _, r := range roots {
			b, err := balanceTx(tx, r)
			if err != nil {
				return err
			}
			balances = append(balances, b)
			remaining += b.Remaining
		}
		remaining = money(remaining)
		if amount > remaining+0.005 {
			return &OverpaymentError{Remaining: remaining}
		}
		// Allocate oldest debt first.
		left := amount
		var note *string
		if n := strings.TrimSpace(req.Note); n != "" {
			note = &n
		}
		for i, r := range roots {
			if left <= 0.005 {
				break
			}
			portion := money(math.Min(left, balances[i].Remaining))
			if portion <= 0 {
				continue
			}
			rootID := r.ID
			p := DocumentPayment{
				ID: idgen.Generate(PrefixPayment), DocumentID: &rootID, StoreID: storeID,
				Amount: portion, Method: method, Note: note, PaidAt: paidAt, CreatedBy: actor.UserID,
			}
			if key != "" {
				p.IdempotencyKey = &key
			}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			left = money(left - portion)
		}
		if err := s.refreshPaymentStateTx(tx, storeID, doc, roots); err != nil {
			return err
		}
		out, err = summaryTx(tx, roots)
		return err
	})
	return out, err
}

// refreshPaymentStateTx sets payment_status on each root's family (and on a
// bill from its roots), completing documents whose debt is settled.
func (s Service) refreshPaymentStateTx(tx *gorm.DB, storeID string, target Document, roots []Document) error {
	allPaid, anyPaid := true, false
	for _, r := range roots {
		b, err := balanceTx(tx, r)
		if err != nil {
			return err
		}
		if b.PaymentStatus != PaymentPaid {
			allPaid = false
		}
		if b.PaymentStatus != PaymentUnpaid {
			anyPaid = true
		}
		var family []Document
		if err := tx.Raw(`WITH RECURSIVE fam AS (
				SELECT id FROM documents WHERE id = ?
				UNION SELECT d.id FROM documents d JOIN fam ON d.source_document_id = fam.id
			)
			SELECT id, type, status FROM documents WHERE id IN (SELECT id FROM fam) AND type <> 'CREDIT_NOTE'`, r.ID).
			Scan(&family).Error; err != nil {
			return err
		}
		if err := applyPaymentState(tx, storeID, family, b.PaymentStatus); err != nil {
			return err
		}
	}
	if target.Type == TypeBill {
		state := PaymentUnpaid
		if allPaid {
			state = PaymentPaid
		} else if anyPaid {
			state = PaymentPartial
		}
		return applyPaymentState(tx, storeID, []Document{target}, state)
	}
	return nil
}

func applyPaymentState(tx *gorm.DB, storeID string, docs []Document, state PaymentStatus) error {
	for _, d := range docs {
		updates := map[string]any{"payment_status": state, "updated_at": time.Now()}
		if state == PaymentPaid && d.Status != StatusCompleted && CanTransition(d.Type, d.Status, StatusCompleted) {
			updates["status"] = StatusCompleted
		}
		if err := tx.Model(&Document{}).Where("id = ? AND store_id = ?", d.ID, storeID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}
