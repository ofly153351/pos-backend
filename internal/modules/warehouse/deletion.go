package warehouse

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pos-backend/internal/platform/lifecycle"
)

// openCountStatuses are the in-progress stock-count session states. A session in any of
// these still actively references its location, so it blocks deletion; completed/cancelled
// sessions are historical only.
var openCountStatuses = []string{"draft", "counting", "review"}

// openReceiptStatuses are the in-progress goods-receipt states (CHECK on
// warehouse_receipts.status). draft/pending_review block deletion; confirmed/cancelled are
// historical.
var openReceiptStatuses = []string{"draft", "pending_review"}

// historicalReceiptStatuses are the terminal goods-receipt states — they are kept purely as
// history and mark a warehouse as "previously used" (archive instead of hard-delete).
var historicalReceiptStatuses = []string{"confirmed", "cancelled"}

// isFKViolation reports whether a database error is a foreign-key RESTRICT violation
// (SQLSTATE 23503) — the residual-race backstop when a dependent row appears between the
// in-memory assessment and the locked delete.
func isFKViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23503")
}

// GatherDeletionBlockers builds a warehouse's dependency snapshot, aggregated across all of
// its non-archived child locations. Read-only.
func (r PostgresRepository) GatherDeletionBlockers(ctx context.Context, storeID, warehouseID string) (lifecycle.BlockerCounts, error) {
	return r.gatherBlockers(r.db.WithContext(ctx), storeID, warehouseID)
}

// gatherBlockers is the shared dependency-counting query set, parameterised by the db handle
// so it runs identically against the base connection (assessment endpoint) and inside the
// locked delete transaction (ApplyDeletion).
func (r PostgresRepository) gatherBlockers(db *gorm.DB, storeID, warehouseID string) (lifecycle.BlockerCounts, error) {
	var b lifecycle.BlockerCounts

	// System protection: the store's default warehouse (stable is_default flag, never the
	// display name).
	var isDefault bool
	if err := db.Table("warehouses").Select("is_default").
		Where("store_id = ? AND id = ?", storeID, warehouseID).Scan(&isDefault).Error; err != nil {
		return b, err
	}

	// Live (non-archived) child location IDs — everything else hangs off these.
	var childIDs []string
	if err := db.Table("locations").
		Where("warehouse_id = ? AND deleted_at IS NULL", warehouseID).
		Pluck("id", &childIDs).Error; err != nil {
		return b, err
	}
	b.LocationCount = len(childIDs)

	// A child sale-point that is the store default makes the whole warehouse protected.
	var protectedChildren int64
	if err := db.Table("locations").
		Where("warehouse_id = ? AND deleted_at IS NULL AND is_default_sale = ?", warehouseID, true).
		Count(&protectedChildren).Error; err != nil {
		return b, err
	}
	b.SystemProtected = isDefault || protectedChildren > 0

	var activeChildren int64
	if err := db.Table("locations").
		Where("warehouse_id = ? AND deleted_at IS NULL AND is_active = ?", warehouseID, true).
		Count(&activeChildren).Error; err != nil {
		return b, err
	}
	b.ActiveLocationCount = int(activeChildren)

	if len(childIDs) > 0 {
		// Stock aggregate across children: SUM(qty) is the hard blocker; row count (even
		// zero-qty rows) is a history marker.
		var agg struct {
			Qty      int64 `gorm:"column:qty"`
			RowCount int64 `gorm:"column:row_count"`
		}
		if err := db.Table("stocks").
			Select("COALESCE(SUM(quantity),0) AS qty, COUNT(*) AS row_count").
			Where("location_id IN ?", childIDs).Scan(&agg).Error; err != nil {
			return b, err
		}
		b.StockQuantity = int(agg.Qty)
		b.StockRowCount = int(agg.RowCount)

		var productDefaults int64
		if err := db.Table("products").
			Where("default_location_id IN ?", childIDs).Count(&productDefaults).Error; err != nil {
			return b, err
		}
		b.ProductDefaultLocationCount = int(productDefaults)

		var movements int64
		if err := db.Table("stock_movements").
			Where("location_id IN ? OR destination_location_id IN ?", childIDs, childIDs).
			Count(&movements).Error; err != nil {
			return b, err
		}
		b.MovementCount = int(movements)

		var openCounts int64
		if err := db.Table("stock_count_sessions").
			Where("location_id IN ? AND status IN ?", childIDs, openCountStatuses).
			Count(&openCounts).Error; err != nil {
			return b, err
		}
		b.OpenStockCountCount = int(openCounts)

		var receiptItems int64
		if err := db.Table("warehouse_receipt_items").
			Where("location_id IN ?", childIDs).Count(&receiptItems).Error; err != nil {
			return b, err
		}
		b.HistoricalReferenceCount += int(receiptItems)

		blocked, err := r.countBlockedChildren(db, childIDs)
		if err != nil {
			return b, err
		}
		b.BlockedChildCount = blocked
	}

	// Open goods receipts addressed to this warehouse (warehouse_receipts.warehouse_id).
	var openReceipts int64
	if err := db.Table("warehouse_receipts").
		Where("warehouse_id = ? AND status IN ?", warehouseID, openReceiptStatuses).
		Count(&openReceipts).Error; err != nil {
		return b, err
	}
	b.OpenReceivingCount = int(openReceipts)

	// Historical references: terminal receipts + cross-store inventory rows.
	var historicalReceipts int64
	if err := db.Table("warehouse_receipts").
		Where("warehouse_id = ? AND status IN ?", warehouseID, historicalReceiptStatuses).
		Count(&historicalReceipts).Error; err != nil {
		return b, err
	}
	b.HistoricalReferenceCount += int(historicalReceipts)

	var inventoryRows int64
	if err := db.Table("warehouse_inventory").
		Where("warehouse_id = ?", warehouseID).Count(&inventoryRows).Error; err != nil {
		return b, err
	}
	b.HistoricalReferenceCount += int(inventoryRows)

	return b, nil
}

// countBlockedChildren counts child locations that individually carry a hard blocker —
// nonzero stock, a product default, default-sale protection, or an open count session. This
// rolls per-location blockers up into the warehouse's BlockedChildCount using the SAME
// blocker semantics as the location decision (no duplicated decision logic).
func (r PostgresRepository) countBlockedChildren(db *gorm.DB, childIDs []string) (int, error) {
	if len(childIDs) == 0 {
		return 0, nil
	}
	blocked := make(map[string]struct{})

	collect := func(q *gorm.DB, col string) error {
		var ids []string
		if err := q.Distinct(col).Pluck(col, &ids).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if id != "" {
				blocked[id] = struct{}{}
			}
		}
		return nil
	}

	if err := collect(db.Table("stocks").Where("location_id IN ? AND quantity <> 0", childIDs), "location_id"); err != nil {
		return 0, err
	}
	if err := collect(db.Table("products").Where("default_location_id IN ?", childIDs), "default_location_id"); err != nil {
		return 0, err
	}
	if err := collect(db.Table("locations").Where("id IN ? AND is_default_sale = ?", childIDs, true), "id"); err != nil {
		return 0, err
	}
	if err := collect(db.Table("stock_count_sessions").Where("location_id IN ? AND status IN ?", childIDs, openCountStatuses), "location_id"); err != nil {
		return 0, err
	}

	return len(blocked), nil
}

// ApplyDeletion performs the smart delete inside a single locked transaction: it locks the
// warehouse row, re-gathers blockers under the lock, re-runs lifecycle.Assess, and applies
// the resolved safe action atomically. See the interface doc for the contract.
func (r PostgresRepository) ApplyDeletion(ctx context.Context, storeID, warehouseID, expected string) (lifecycle.Assessment, string, error) {
	var assessment lifecycle.Assessment
	var applied string

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the target row; capture deleted_at for the idempotency check.
		var head struct {
			DeletedAt *time.Time `gorm:"column:deleted_at"`
		}
		res := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("warehouses").Select("deleted_at").
			Where("store_id = ? AND id = ?", storeID, warehouseID).
			Scan(&head)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrWarehouseNotFound
		}
		// Idempotent double-click: already archived → success no-op. Leave the assessment at
		// its zero value (no determination) rather than synthesizing one from empty blockers:
		// once archived, re-gathering would see zero ACTIVE children (deleted_at filter) and
		// lose sight of the preserved movement history, misleadingly reading as "hard_delete".
		// The authoritative signal for the caller is the returned action ("archived").
		if head.DeletedAt != nil {
			applied = "archived"
			return nil
		}

		b, err := r.gatherBlockers(tx, storeID, warehouseID)
		if err != nil {
			return err
		}
		a := lifecycle.Assess(lifecycle.EntityWarehouse, b)
		assessment = a

		// Optimistic concurrency: the client may declare the action it last saw. If the
		// locked re-assessment disagrees, abort so the user re-confirms against fresh state.
		if expected != "" && expected != a.SuggestedAction {
			return ErrEntityStateChanged
		}

		now := time.Now().UTC()
		switch a.SuggestedAction {
		case lifecycle.ActionArchive:
			if err := tx.Table("locations").
				Where("warehouse_id = ? AND deleted_at IS NULL", warehouseID).
				Updates(map[string]any{"deleted_at": now, "is_active": false, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Table("warehouses").
				Where("store_id = ? AND id = ?", storeID, warehouseID).
				Updates(map[string]any{"deleted_at": now, "is_active": false, "updated_at": now}).Error; err != nil {
				return err
			}
			applied = "archived"

		case lifecycle.ActionHardDelete:
			// Never-used warehouse: drop its (reference-free) child locations then the
			// warehouse, atomically. A RESTRICT violation means a dependent row raced in —
			// roll back and report in-use.
			if err := tx.Exec("DELETE FROM locations WHERE warehouse_id = ?", warehouseID).Error; err != nil {
				if isFKViolation(err) {
					return ErrWarehouseInUse
				}
				return err
			}
			del := tx.Where("store_id = ? AND id = ?", storeID, warehouseID).Delete(&Warehouse{})
			if del.Error != nil {
				if isFKViolation(del.Error) {
					return ErrWarehouseInUse
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
	case lifecycle.CodeWarehouseSystemProtected:
		return ErrDefaultWarehouseDelete
	case lifecycle.CodeWarehouseHasStock:
		return ErrWarehouseHasStock
	case lifecycle.CodeWarehouseHasBlockedLocations:
		return ErrWarehouseHasBlockedLocations
	case lifecycle.CodeWarehouseHasOpenOperations:
		return ErrWarehouseHasOpenOperations
	default:
		return ErrWarehouseInUse
	}
}
