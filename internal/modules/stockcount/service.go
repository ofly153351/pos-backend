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
		var status string
		if err := tx.Table("stock_count_sessions").
			Select("status").
			Where("id = ? AND store_id = ?", sessionID, storeID).
			Take(&status).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSessionNotFound
			}
			return err
		}
		if status == "completed" {
			return ErrSessionAlreadyApplied
		}

		// Bind the adjustment service to THIS transaction so every movement + stock
		// write participates in the same atomic unit. Any error returned here aborts
		// the closure and GORM rolls the whole batch back.
		mv := stock_movement.NewService(stock_movement.NewPostgresRepository(tx), tx)
		for _, it := range req.Items {
			if strings.TrimSpace(it.ProductID) == "" {
				return ErrInvalidApplyItem
			}
			if _, err := mv.AdjustStock(ctx, actor, storeID, stock_movement.AdjustStockRequest{
				ProductID:    it.ProductID,
				PhysicalQty:  it.CountedQty,
				Note:         it.Note,
				ReferenceID:  sessionID,
				MovementType: stock_movement.MovementTypeCountCorrection,
			}); err != nil {
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
