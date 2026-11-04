package document

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pos-backend/internal/platform/bizdate"
)

// insertNumberedTx assigns the next per-store/type/month document number and
// inserts doc inside tx. A transaction-scoped advisory lock on (store, prefix)
// serializes numbering, so no insert ever fails on a duplicate number — which
// matters because a failed statement aborts the surrounding Postgres
// transaction and could not be retried in place (B07/B12).
func insertNumberedTx(tx *gorm.DB, doc *Document) error {
	now := bizdate.Local(time.Now())
	prefix := fmt.Sprintf("%s%d%02d", typePrefix(doc.Type), now.Year()+543, now.Month())
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))", doc.StoreID, prefix).Error; err != nil {
		return err
	}
	var maxSeq int64
	if err := tx.Model(&Document{}).
		Where("store_id = ? AND document_no LIKE ?", doc.StoreID, prefix+"-%").
		Select("COALESCE(MAX(CAST(SUBSTRING(document_no FROM '[0-9]+$') AS INTEGER)), 0)").
		Scan(&maxSeq).Error; err != nil {
		return err
	}
	doc.DocumentNo = fmt.Sprintf("%s-%04d", prefix, maxSeq+1)
	doc.DocumentNoFull = doc.DocumentNo
	return tx.Create(doc).Error
}

// lockDocumentTx loads and row-locks a store document for the rest of tx.
func lockDocumentTx(tx *gorm.DB, storeID, id string) (Document, error) {
	var doc Document
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&doc, "id = ? AND store_id = ?", id, storeID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return doc, ErrNotFound
	}
	return doc, err
}

// activeChildTx returns an existing non-cancelled document of type target that
// was created from sourceID (directly, or for BILL through the delivery-order
// link table). Cancelled children do not block a new conversion.
func activeChildTx(tx *gorm.DB, sourceID string, target DocumentType) (*Document, error) {
	var child Document
	q := tx.Where("type = ? AND status <> ?", target, StatusCancelled)
	if target == TypeBill {
		q = q.Where("source_document_id = ? OR id IN (SELECT bill_document_id FROM bill_delivery_order_links WHERE delivery_order_id = ?)", sourceID, sourceID)
	} else {
		q = q.Where("source_document_id = ?", sourceID)
	}
	err := q.Order("created_at").First(&child).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &child, nil
}

// SourceNotEligibleError names the closed source document and its status.
type SourceNotEligibleError struct {
	DocumentNo string
	Status     DocumentStatus
}

func (e *SourceNotEligibleError) Error() string {
	return fmt.Sprintf("source document %s is %s", e.DocumentNo, e.Status)
}

func (e *SourceNotEligibleError) Unwrap() error { return ErrSourceNotEligible }

func sourceNotEligible(src Document) error {
	return &SourceNotEligibleError{DocumentNo: src.DocumentNoFull, Status: src.Status}
}
