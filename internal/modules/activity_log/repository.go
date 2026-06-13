package activity_log

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/idgen"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) Create(ctx context.Context, log ActivityLog) error {
	log.ID = idgen.Generate(idgen.PrefixActivityLog)
	log.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(&log).Error
}

// UserCanOperateStore reports whether the user is a member (owner/manager/cashier)
// of the store, or a platform admin. Mirrors the membership check used across the
// other modules so the audit trail cannot be read cross-tenant.
func (r Repository) UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager", "cashier"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r Repository) List(ctx context.Context, q ListQuery) ([]ActivityLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&ActivityLog{}).Where("store_id = ?", q.StoreID)

	if q.Module != "" {
		query = query.Where("module = ?", q.Module)
	}
	if q.Action != "" {
		query = query.Where("action = ?", q.Action)
	}
	if q.UserID != "" {
		query = query.Where("user_id = ?", q.UserID)
	}
	if q.DateFrom != "" {
		query = query.Where("created_at >= ?", q.DateFrom)
	}
	if q.DateTo != "" {
		query = query.Where("created_at < ?", q.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (q.Page - 1) * q.Limit
	var logs []ActivityLog
	err := query.Order("created_at DESC").Offset(offset).Limit(q.Limit).Find(&logs).Error
	return logs, total, err
}
