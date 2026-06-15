package stockcount

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
	List(ctx context.Context, storeID string) ([]CountSession, error)
	Get(ctx context.Context, storeID, sessionID string) (CountSession, error)
	Upsert(ctx context.Context, session CountSession) error
	Delete(ctx context.Context, storeID, sessionID string) (int64, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ? AND status <> 'suspended'", storeID, userID, []string{"owner", "manager", "cashier"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r PostgresRepository) loadItems(ctx context.Context, sessionIDs []string) (map[string][]CountItem, error) {
	out := make(map[string][]CountItem)
	if len(sessionIDs) == 0 {
		return out, nil
	}
	var items []CountItem
	err := r.db.WithContext(ctx).
		Table("stock_count_items").
		Where("session_id IN ?", sessionIDs).
		Order("sort_order ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		out[it.SessionID] = append(out[it.SessionID], it)
	}
	return out, nil
}

func (r PostgresRepository) List(ctx context.Context, storeID string) ([]CountSession, error) {
	var sessions []CountSession
	err := r.db.WithContext(ctx).
		Table("stock_count_sessions").
		Where("store_id = ?", storeID).
		Order("created_at DESC").
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	byID, err := r.loadItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		if v := byID[sessions[i].ID]; v != nil {
			sessions[i].Items = v
		} else {
			sessions[i].Items = []CountItem{}
		}
	}
	return sessions, nil
}

func (r PostgresRepository) Get(ctx context.Context, storeID, sessionID string) (CountSession, error) {
	var s CountSession
	err := r.db.WithContext(ctx).
		Table("stock_count_sessions").
		Where("store_id = ? AND id = ?", storeID, sessionID).
		Take(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CountSession{}, ErrSessionNotFound
		}
		return CountSession{}, err
	}
	byID, err := r.loadItems(ctx, []string{s.ID})
	if err != nil {
		return CountSession{}, err
	}
	if v := byID[s.ID]; v != nil {
		s.Items = v
	} else {
		s.Items = []CountItem{}
	}
	return s, nil
}

// sessionHeader returns the mutable header columns (a map so GORM writes NULLs
// for cleared nullable fields rather than skipping zero values).
func sessionHeader(s CountSession) map[string]any {
	return map[string]any{
		"name":           s.Name,
		"warehouse_name": s.WarehouseName,
		"zone":           s.Zone,
		"category_id":    s.CategoryID,
		"category_name":  s.CategoryName,
		"staff":          s.Staff,
		"note":           s.Note,
		"status":         s.Status,
		"count_type":     s.CountType,
		"cycle_rule":     s.CycleRule,
		"blind_count":    s.BlindCount,
		"completed_by":   s.CompletedBy,
		"completed_at":   s.CompletedAt,
		"updated_at":     s.UpdatedAt,
	}
}

func itemRows(s CountSession) []map[string]any {
	rows := make([]map[string]any, 0, len(s.Items))
	for i, it := range s.Items {
		rows = append(rows, map[string]any{
			"id":                    newItemID(),
			"session_id":            s.ID,
			"product_id":            it.ProductID,
			"name":                  it.Name,
			"sku":                   it.SKU,
			"barcode":               it.Barcode,
			"system_qty":            it.SystemQty,
			"min_stock":             it.MinStock,
			"location":              it.Location,
			"counted":               it.Counted,
			"note":                  it.Note,
			"skipped":               it.Skipped,
			"variance_reason":       it.VarianceReason,
			"variance_reason_other": it.VarianceReasonOther,
			"count_user":            it.CountUser,
			"counted_at":            it.CountedAt,
			"cost_basis":            it.CostBasis,
			"adjusted":              it.Adjusted,
			"adjusted_at":           it.AdjustedAt,
			"adjusted_by":           it.AdjustedBy,
			"sort_order":            i,
		})
	}
	return rows
}

// Upsert writes the session header (insert or update) and fully replaces its
// items, in one transaction — the worksheet is saved as a single document.
func (r PostgresRepository) Upsert(ctx context.Context, s CountSession) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	res := tx.Table("stock_count_sessions").
		Where("id = ? AND store_id = ?", s.ID, s.StoreID).
		Updates(sessionHeader(s))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		insert := sessionHeader(s)
		insert["id"] = s.ID
		insert["store_id"] = s.StoreID
		insert["created_by"] = s.CreatedBy
		insert["created_at"] = s.CreatedAt
		if err := tx.Table("stock_count_sessions").Create(insert).Error; err != nil {
			return err
		}
	}

	if err := tx.Exec("DELETE FROM stock_count_items WHERE session_id = ?", s.ID).Error; err != nil {
		return err
	}
	if rows := itemRows(s); len(rows) > 0 {
		if err := tx.Table("stock_count_items").Create(rows).Error; err != nil {
			return err
		}
	}
	return tx.Commit().Error
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, sessionID string) (int64, error) {
	res := r.db.WithContext(ctx).
		Exec("DELETE FROM stock_count_sessions WHERE store_id = ? AND id = ?", storeID, sessionID)
	return res.RowsAffected, res.Error
}
