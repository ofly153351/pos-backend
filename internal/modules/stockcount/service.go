package stockcount

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/stock_movement"
)

type Service struct {
	repo Repository
	db   *gorm.DB
}

func NewService(repo Repository, db *gorm.DB) Service {
	return Service{repo: repo, db: db}
}

// ── Access ──────────────────────────────────────────────────────────────────
// Counting is an operational task; any store member (owner/manager/cashier) may
// create, continue and apply counts — mirroring the existing stock-movement model.

func (s Service) ensureAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrStoreIDRequired
	}
	ok, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// validateCountLocation confirms a location exists, belongs to THIS store and is active —
// the same store+active fence the stock-movement explicit-location path uses. A count may
// only target an active location in its own store.
func (s Service) validateCountLocation(ctx context.Context, storeID, locationID string) error {
	var cnt int64
	if err := s.db.WithContext(ctx).
		Table("locations").
		Where("id = ? AND store_id = ? AND is_active = TRUE", locationID, storeID).
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return ErrCountLocationInvalid
	}
	return nil
}

func (s Service) List(ctx context.Context, actor auth.Claims, storeID string) ([]CountSession, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, storeID)
}

func (s Service) Get(ctx context.Context, actor auth.Claims, storeID, sessionID string) (CountSession, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return CountSession{}, err
	}
	return s.repo.Get(ctx, storeID, sessionID)
}

// Save upserts a single session (header + items) addressed by the URL sessionID.
func (s Service) Save(ctx context.Context, actor auth.Claims, storeID, sessionID string, session CountSession) (CountSession, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return CountSession{}, err
	}
	if strings.TrimSpace(sessionID) == "" {
		return CountSession{}, ErrSessionIDRequired
	}
	if strings.TrimSpace(session.ID) == "" {
		session.ID = sessionID
	}
	if session.ID != sessionID {
		return CountSession{}, ErrSessionIDMismatch
	}

	session.StoreID = storeID
	// A provided location must be an active location in this store. (Presence is required to
	// APPLY — enforced in Apply — but a draft may be saved while the location is chosen.)
	if session.LocationID != nil && strings.TrimSpace(*session.LocationID) != "" {
		if err := s.validateCountLocation(ctx, storeID, strings.TrimSpace(*session.LocationID)); err != nil {
			return CountSession{}, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	session.UpdatedAt = now
	if strings.TrimSpace(session.CreatedAt) == "" {
		session.CreatedAt = now
	}
	if strings.TrimSpace(session.CreatedBy) == "" {
		session.CreatedBy = actor.UserID
	}
	if session.Items == nil {
		session.Items = []CountItem{}
	}

	if err := s.repo.Upsert(ctx, session); err != nil {
		return CountSession{}, err
	}
	return s.repo.Get(ctx, storeID, sessionID)
}

// Apply commits the counted quantities as real stock corrections — atomically.
// Every item's COUNT_CORRECTION movement, the stock write, the per-item "adjusted"
// flag and the session completion all run inside ONE transaction, so a failure on
// any single item rolls the whole batch back (fail-all) and leaves the session in
// review. It reuses the proven stock_movement.AdjustStock logic via a
// transaction-scoped service, so behaviour matches a normal manual adjustment.
func (s Service) Apply(ctx context.Context, actor auth.Claims, storeID, sessionID string, req ApplyRequest) (CountSession, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return CountSession{}, err
	}
	if strings.TrimSpace(sessionID) == "" {
		return CountSession{}, ErrSessionIDRequired
	}
	if len(req.Items) == 0 {
		return CountSession{}, ErrNoItemsToApply
	}

	now := time.Now().UTC().Format(time.RFC3339)
	productIDs := make([]string, 0, len(req.Items))

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The session must exist, belong to the store, and not already be applied.
		var hdr struct {
			Status     string  `gorm:"column:status"`
			LocationID *string `gorm:"column:location_id"`
		}
		if err := tx.Table("stock_count_sessions").
			Select("status, location_id").
			Where("id = ? AND store_id = ?", sessionID, storeID).
			Take(&hdr).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSessionNotFound
			}
			return err
		}
		if hdr.Status == "completed" {
			return ErrSessionAlreadyApplied
		}
		// Location-scoped apply: the session must target exactly one active location, and EVERY
		// correction is pinned to it. A legacy session without a location cannot be applied (its
		// system quantities are aggregates that would be unsafe to write into one location).
		if hdr.LocationID == nil || strings.TrimSpace(*hdr.LocationID) == "" {
			return ErrCountLocationRequired
		}
		sessionLocationID := strings.TrimSpace(*hdr.LocationID)
		if err := s.validateCountLocation(ctx, storeID, sessionLocationID); err != nil {
			return err
		}

		// The system quantity captured for each worksheet row at count time. It is the
		// optimistic-lock "expected" for the SET_ACTUAL apply: AdjustStock rejects the
		// write unless this matches the row-locked LOCATION quantity. For a single-location
		// product system_qty == that location's on-hand, so a genuine correction applies;
		// for a multi-location product system_qty is the store-wide total which never
		// equals one location's quantity, so the apply is safely blocked (ErrStockStaleCount)
		// instead of inflating one location — and a count made stale by an intervening
		// sale/transfer is caught the same way. Read inside the tx so it is authoritative
		// (not trusted from the client payload).
		var sysRows []struct {
			ProductID string `gorm:"column:product_id"`
			SystemQty int    `gorm:"column:system_qty"`
		}
		if err := tx.Table("stock_count_items").
			Select("product_id, system_qty").
			Where("session_id = ?", sessionID).
			Find(&sysRows).Error; err != nil {
			return err
		}
		systemQty := make(map[string]int, len(sysRows))
		for _, r := range sysRows {
			systemQty[r.ProductID] = r.SystemQty
		}

		// Bind the adjustment service to THIS transaction so every movement + stock
		// write participates in the same atomic unit. Any error returned here aborts
		// the closure and GORM rolls the whole batch back.
		mv := stock_movement.NewService(stock_movement.NewPostgresRepository(tx), tx)
		for _, it := range req.Items {
			if strings.TrimSpace(it.ProductID) == "" {
				return ErrInvalidApplyItem
			}
			expected := systemQty[it.ProductID]
			// No variance (counted == expected) → nothing to write; still stamp the row as
			// applied below. Skipping avoids a needless optimistic-lock comparison.
			if it.CountedQty == expected {
				productIDs = append(productIDs, it.ProductID)
				continue
			}
			if _, err := mv.AdjustStock(ctx, actor, storeID, stock_movement.AdjustStockRequest{
				ProductID:    it.ProductID,
				LocationID:   sessionLocationID, // pin the correction to the session's location
				PhysicalQty:  it.CountedQty,
				ExpectedQty:  &expected,         // optimistic lock vs THIS location's live qty — see comment above
				Reason:       "DATA_CORRECTION", // count correction reason (W2 SET_ACTUAL set)
				Note:         it.Note,
				ReferenceID:  sessionID,
				MovementType: stock_movement.MovementTypeCountCorrection,
			}); err != nil && !errors.Is(err, stock_movement.ErrStockNoChange) {
				return err
			}
			productIDs = append(productIDs, it.ProductID)
		}

		// Stamp the applied worksheet rows so the audit trail reflects reality.
		if err := tx.Table("stock_count_items").
			Where("session_id = ? AND product_id IN ?", sessionID, productIDs).
			Updates(map[string]any{
				"adjusted":    true,
				"adjusted_at": now,
				"adjusted_by": actor.UserID,
			}).Error; err != nil {
			return err
		}

		// Complete the session.
		return tx.Table("stock_count_sessions").
			Where("id = ? AND store_id = ?", sessionID, storeID).
			Updates(map[string]any{
				"status":       "completed",
				"completed_at": now,
				"completed_by": actor.UserID,
				"updated_at":   now,
			}).Error
	})
	if err != nil {
		return CountSession{}, err
	}

	return s.repo.Get(ctx, storeID, sessionID)
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, sessionID string) error {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return err
	}
	affected, err := s.repo.Delete(ctx, storeID, sessionID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSessionNotFound
	}
	return nil
}
