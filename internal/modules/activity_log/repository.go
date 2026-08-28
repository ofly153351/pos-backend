package activity_log

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/idgen"
)

// applyPairs narrows a query to rows whose (module, action) matches one of the
// supplied tuples. An empty set means the requested severity/category does not map
// to any known activity → return zero rows (1=0) rather than silently ignoring it.
func applyPairs(query *gorm.DB, pairs [][2]string) *gorm.DB {
	if len(pairs) == 0 {
		return query.Where("1 = 0")
	}
	clauses := make([]string, 0, len(pairs))
	args := make([]any, 0, len(pairs)*2)
	for _, p := range pairs {
		clauses = append(clauses, "(module = ? AND action = ?)")
		args = append(args, p[0], p[1])
	}
	return query.Where(strings.Join(clauses, " OR "), args...)
}

// parseFilterDate leniently accepts the date formats the UI sends (a bare date, an
// RFC3339 timestamp, or a local datetime). Unparseable input is ignored rather than
// passed raw into SQL.
func parseFilterDate(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

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
	if q.ResourceID != "" {
		query = query.Where("resource_id = ?", q.ResourceID)
	}
	// Severity / Category are derived, never stored. Translate them into concrete
	// (module, action) predicates generated from the same taxonomy the DTO uses, so
	// filtering at scale stays a plain indexed WHERE and can never drift from the badge.
	if q.Severity != "" {
		query = applyPairs(query, SeverityPairs(q.Severity))
	}
	if q.Category != "" {
		query = applyPairs(query, CategoryPairs(q.Category))
	}
	// Free-text search — deliberately limited to business-facing columns. Never
	// path/ip/session ids (per spec: owners search by product, user, action — not IDs).
	if s := strings.TrimSpace(q.Search); s != "" {
		like := "%" + s + "%"
		query = query.Where(
			"user_name ILIKE ? OR module ILIKE ? OR action ILIKE ? OR resource_id ILIKE ?",
			like, like, like, like,
		)
	}
	if from, ok := parseFilterDate(q.DateFrom); ok {
		query = query.Where("created_at >= ?", from)
	}
	if to, ok := parseFilterDate(q.DateTo); ok {
		query = query.Where("created_at < ?", to)
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
