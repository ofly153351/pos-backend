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
