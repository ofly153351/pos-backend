package document

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// OutstandingDebt is one open obligation under the shared debt rule (B09,
// B11): one row per payable root, never per derived document.
type OutstandingDebt struct {
	ID             string     `gorm:"column:id" json:"id"`
	DocumentNoFull string     `gorm:"column:document_no_full" json:"document_no"`
	Type           string     `gorm:"column:type" json:"type"`
	Status         string     `gorm:"column:status" json:"status"`
	CustomerID     string     `gorm:"column:customer_id" json:"customer_id"`
	DocumentDate   time.Time  `gorm:"column:document_date" json:"document_date"`
	DueDate        *time.Time `gorm:"column:due_date" json:"due_date,omitempty"`
	Total          float64    `gorm:"column:total" json:"total"`
	Paid           float64    `gorm:"column:paid" json:"paid"`
	Credited       float64    `gorm:"column:credited" json:"credited"`
	Remaining      float64    `gorm:"column:remaining" json:"remaining"`
}

// outstandingSQL mirrors payableRootsTx/balanceTx in SQL:
//   - every document walks up source_document_id until it reaches an INVOICE
//     (quotations are never debts), and that INVOICE — or the top-most
//     document when there is none — is its root;
//   - a BILL with linked delivery orders is never a root (its delivery orders'
//     roots carry the debt);
//   - remaining = root total − ledger payments − credit notes in its family.
//
// Only the store's documents are scanned. Cancelled/closed roots are excluded.
const outstandingSQL = `
WITH RECURSIVE chain AS (
	SELECT d.id AS doc_id, d.id AS node_id, d.type AS node_type, d.source_document_id AS parent_id, 0 AS depth
	FROM documents d WHERE d.store_id = @store
	UNION ALL
	SELECT c.doc_id, p.id, p.type, p.source_document_id, c.depth + 1
	FROM chain c JOIN documents p ON p.id = c.parent_id
	WHERE c.node_type <> 'INVOICE' AND p.type <> 'QUOTATION' AND c.depth < 8
),
root_of AS (
	SELECT DISTINCT ON (doc_id) doc_id, node_id AS root_id
	FROM chain ORDER BY doc_id, depth DESC
),
paid AS (
	SELECT document_id AS root_id, SUM(paid_amount) AS paid
	FROM invoice_payments WHERE document_id IS NOT NULL AND NOT is_voided
	GROUP BY document_id
),
credited AS (
	SELECT r.root_id, SUM(cn.total_amount) AS credited
	FROM documents cn JOIN root_of r ON r.doc_id = cn.source_document_id
	WHERE cn.store_id = @store AND cn.type = 'CREDIT_NOTE' AND cn.status <> 'CANCELLED'
	GROUP BY r.root_id
)
SELECT d.id, d.document_no_full, d.type, d.status, d.customer_id, d.document_date, d.due_date,
	d.total_amount AS total,
	COALESCE(p.paid, 0) AS paid,
	COALESCE(c.credited, 0) AS credited,
	GREATEST(d.total_amount - COALESCE(p.paid, 0) - COALESCE(c.credited, 0), 0) AS remaining
FROM documents d
JOIN root_of r ON r.doc_id = d.id AND r.root_id = d.id
LEFT JOIN paid p ON p.root_id = d.id
LEFT JOIN credited c ON c.root_id = d.id
WHERE d.store_id = @store
	AND d.type IN ('INVOICE','BILL','DELIVERY_ORDER','TAX_INVOICE')
	AND d.status NOT IN ('CANCELLED','REJECTED','EXPIRED','CONVERTED')
	AND NOT (d.type = 'BILL' AND EXISTS (SELECT 1 FROM bill_delivery_order_links l WHERE l.bill_document_id = d.id))
	AND d.total_amount - COALESCE(p.paid, 0) - COALESCE(c.credited, 0) > 0.005`

// OutstandingFilter narrows the outstanding set (empty = no filter).
type OutstandingFilter struct {
	CustomerID string
	DateFrom   string // YYYY-MM-DD business date, inclusive
	DateTo     string
}

// outstandingDebts returns open obligations for a store.
func outstandingDebts(ctx context.Context, db *gorm.DB, storeID string, f OutstandingFilter) ([]OutstandingDebt, error) {
	q := "SELECT * FROM (" + outstandingSQL + ") o WHERE 1=1"
	args := map[string]any{"store": storeID}
	if f.CustomerID != "" {
		q += " AND o.customer_id = @customer"
		args["customer"] = f.CustomerID
	}
	if f.DateFrom != "" {
		q += " AND (o.document_date AT TIME ZONE 'Asia/Bangkok')::date >= CAST(@date_start AS date)"
		args["date_start"] = f.DateFrom
	}
	if f.DateTo != "" {
		q += " AND (o.document_date AT TIME ZONE 'Asia/Bangkok')::date <= CAST(@date_end AS date)"
		args["date_end"] = f.DateTo
	}
	q += " ORDER BY o.document_date, o.id"
	var rows []OutstandingDebt
	err := db.WithContext(ctx).Raw(q, args).Scan(&rows).Error
	return rows, err
}
