package promotion

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	List(ctx context.Context, storeID string) ([]json.RawMessage, error)
	GetData(ctx context.Context, storeID, id string) (string, error)
	Insert(ctx context.Context, row PromotionRow) error
	Update(ctx context.Context, storeID, id string, updates map[string]any) (int64, error)
	Delete(ctx context.Context, storeID, id string) (int64, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) List(ctx context.Context, storeID string) ([]json.RawMessage, error) {
	var rows []struct {
		Data               string  `gorm:"column:data"`
		UsageCount         int64   `gorm:"column:usage_count"`
		DiscountGivenTotal float64 `gorm:"column:discount_given_total"`
	}
	err := r.db.WithContext(ctx).
		Table("promotions").
		Select("data, usage_count, discount_given_total").
		Where("store_id = ? AND deleted_at IS NULL", storeID).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		// Overlay the server-tracked analytics onto the stored campaign JSON so the
		// UI shows real usage/discount instead of the client-time 0 placeholders.
		var obj map[string]any
		if json.Unmarshal([]byte(row.Data), &obj) == nil {
			obj["usageCount"] = row.UsageCount
			obj["discountGiven"] = row.DiscountGivenTotal
			if merged, mErr := json.Marshal(obj); mErr == nil {
				out[i] = json.RawMessage(merged)
				continue
			}
		}
		out[i] = json.RawMessage(row.Data)
	}
	return out, nil
}

// GetData returns the stored campaign JSON for a promotion, or "" if it does not
// exist. Used to snapshot the prior state for the Activity Center change log.
func (r PostgresRepository) GetData(ctx context.Context, storeID, id string) (string, error) {
	var data string
	err := r.db.WithContext(ctx).
		Table("promotions").
		Select("data").
		Where("store_id = ? AND id = ? AND deleted_at IS NULL", storeID, id).
		Limit(1).
		Scan(&data).Error
	return data, err
}

func (r PostgresRepository) Insert(ctx context.Context, row PromotionRow) error {
	return r.db.WithContext(ctx).Table("promotions").Create(map[string]any{
		"id":         row.ID,
		"store_id":   row.StoreID,
		"name":       row.Name,
		"type":       row.Type,
		"status":     row.Status,
		"code":       row.Code,
		"data":       row.Data,
		"created_by": row.CreatedBy,
		"created_at": row.CreatedAt,
		"updated_at": row.UpdatedAt,
	}).Error
}

func (r PostgresRepository) Update(ctx context.Context, storeID, id string, updates map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).
		Table("promotions").
		Where("store_id = ? AND id = ? AND deleted_at IS NULL", storeID, id).
		Updates(updates)
	return res.RowsAffected, res.Error
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, id string) (int64, error) {
	// Soft delete: hide from active lists + checkout verification while preserving
	// the row so historical sales / usage rows keep a valid reference.
	now := time.Now().UTC().Format(time.RFC3339)
	res := r.db.WithContext(ctx).Exec(
		"UPDATE promotions SET deleted_at = ?, updated_at = ? WHERE store_id = ? AND id = ? AND deleted_at IS NULL",
		now, now, storeID, id,
	)
	return res.RowsAffected, res.Error
}
