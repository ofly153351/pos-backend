package location

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pos-backend/internal/platform/lifecycle"
)

// openCountStatuses are the in-progress stock-count session states — a session in any of
// these still actively references its location and blocks deletion; completed/cancelled are
// historical only.
var openCountStatuses = []string{"draft", "counting", "review"}

// openReceiptStatuses are the in-progress goods-receipt states (CHECK on
// warehouse_receipts.status). A receipt line on this location whose parent receipt is in one
// of these states is an open operation; confirmed/cancelled lines are historical.
var openReceiptStatuses = []string{"draft", "pending_review"}

// isFKViolation reports whether a database error is a foreign-key RESTRICT violation
// (SQLSTATE 23503) — the residual-race backstop when a dependent row appears between the
// in-memory assessment and the locked delete.
func isFKViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23503")
}

// GatherDeletionBlockers builds a single location's dependency snapshot. Read-only.
func (r PostgresRepository) GatherDeletionBlockers(ctx context.Context, storeID, locationID string) (lifecycle.BlockerCounts, error) {
	return r.gatherBlockers(r.db.WithContext(ctx), storeID, locationID)
}

// gatherBlockers is the shared dependency-counting query set, parameterised by the db handle
// so it runs identically against the base connection (assessment endpoint) and inside the
// locked delete transaction (ApplyDeletion).
func (r PostgresRepository) gatherBlockers(db *gorm.DB, storeID, locationID string) (lifecycle.BlockerCounts, error) {
	var b lifecycle.BlockerCounts

	// System protection: the store's default sale location (stable is_default_sale flag,
	// never the display name). is_sale_point alone is NOT protection — a non-default sale
	// point can still be archived.
	var isDefaultSale bool
	if err := db.Table("locations").Select("is_default_sale").
		Where("store_id = ? AND id = ?", storeID, locationID).Scan(&isDefaultSale).Error; err != nil {
		return b, err
	}
	b.SystemProtected = isDefaultSale

	// Stock: SUM(qty) is the hard blocker; row count (even zero-qty rows) is a history marker.
	var agg struct {
		Qty      int64 `gorm:"column:qty"`
		RowCount int64 `gorm:"column:row_count"`
	}
	if err := db.Table("stocks").
		Select("COALESCE(SUM(quantity),0) AS qty, COUNT(*) AS row_count").
		Where("location_id = ?", locationID).Scan(&agg).Error; err != nil {
		return b, err
	}
	b.StockQuantity = int(agg.Qty)
	b.StockRowCount = int(agg.RowCount)

	var productDefaults int64
	if err := db.Table("products").
		Where("default_location_id = ?", locationID).Count(&productDefaults).Error; err != nil {
		return b, err
	}
	b.ProductDefaultLocationCount = int(productDefaults)

	var movements int64
	if err := db.Table("stock_movements").
		Where("location_id = ? OR destination_location_id = ?", locationID, locationID).
		Count(&movements).Error; err != nil {
		return b, err
	}
	b.MovementCount = int(movements)

	var openCounts int64
	if err := db.Table("stock_count_sessions").
		Where("location_id = ? AND status IN ?", locationID, openCountStatuses).
		Count(&openCounts).Error; err != nil {
		return b, err
	}
	b.OpenStockCountCount = int(openCounts)

	// Open goods receipts: receipt lines on this location whose parent receipt is still open.
	var openReceipts int64
	if err := db.Table("warehouse_receipt_items AS wri").
		Joins("JOIN warehouse_receipts AS wr ON wr.id = wri.receipt_id").
		Where("wri.location_id = ? AND wr.status IN ?", locationID, openReceiptStatuses).
		Count(&openReceipts).Error; err != nil {
		return b, err
	}
	b.OpenReceivingCount = int(openReceipts)

	// Historical references: every receipt line that ever pointed at this location.
	var receiptItems int64
	if err := db.Table("warehouse_receipt_items").
		Where("location_id = ?", locationID).Count(&receiptItems).Error; err != nil {
		return b, err
	}
	b.HistoricalReferenceCount += int(receiptItems)

	return b, nil
}

// ApplyDeletion performs the smart delete inside a single locked transaction: it locks the
// location row, re-gathers blockers under the lock, re-runs lifecycle.Assess, and applies
// the resolved safe action atomically. See the interface doc for the contract.
func (r PostgresRepository) ApplyDeletion(ctx context.Context, storeID, locationID, expected string) (lifecycle.Assessment, string, error) {
	var assessment lifecycle.Assessment
	var applied string

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the target row; capture deleted_at for the idempotency check.
		var head struct {
			DeletedAt *time.Time `gorm:"column:deleted_at"`
		}
		res := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("locations").Select("deleted_at").
			Where("store_id = ? AND id = ?", storeID, locationID).
			Scan(&head)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrLocationNotFound
		}
		// Idempotent double-click: already archived → success no-op. Leave the assessment at
		// its zero value (no determination) rather than synthesizing one from empty blockers,
		// which would misleadingly read as "hard_delete". The authoritative signal for the
		// caller is the returned action ("archived").
		if head.DeletedAt != nil {
			applied = "archived"
			return nil
		}

		b, err := r.gatherBlockers(tx, storeID, locationID)
		if err != nil {
			return err
		}
		a := lifecycle.Assess(lifecycle.EntityLocation, b)
		assessment = a

		// Optimistic concurrency: the client may declare the action it last saw. If the
		// locked re-assessment disagrees, abort so the user re-confirms against fresh state.
		if expected != "" && expected != a.SuggestedAction {
			return ErrLocationStateChanged
		}

		now := time.Now().UTC()
		switch a.SuggestedAction {
		case lifecycle.ActionArchive:
			if err := tx.Table("locations").
				Where("store_id = ? AND id = ?", storeID, locationID).
				Updates(map[string]any{"deleted_at": now, "is_active": false, "updated_at": now}).Error; err != nil {
				return err
			}
			applied = "archived"

		case lifecycle.ActionHardDelete:
			// Never-used location: a plain delete. A RESTRICT violation means a dependent
			// row raced in — roll back and report in-use.
			del := tx.Where("store_id = ? AND id = ?", storeID, locationID).Delete(&Location{})
			if del.Error != nil {
				if isFKViolation(del.Error) {
					return ErrLocationInUse
				}
				return del.Error
			}
			applied = "deleted"

		default: // blocked
			return blockerError(a.BlockerCode)
		}
		return nil
	})
	if err != nil {
		return assessment, "", err
	}
	return assessment, applied, nil
}

// blockerError maps a lifecycle blocker code to the module's localized sentinel error, so
// the HTTP layer returns the right Thai message while the machine code travels in the
// structured error details.
func blockerError(code string) error {
	switch code {
	case lifecycle.CodeLocationSystemProtected:
		return ErrDefaultSaleLocationDelete
	case lifecycle.CodeLocationHasStock:
		return ErrLocationHasStock
	case lifecycle.CodeLocationIsProductDefault:
		return ErrLocationIsProductDefault
	case lifecycle.CodeLocationHasOpenOperations:
		return ErrLocationHasOpenOperations
	default:
		return ErrLocationInUse
	}
}
