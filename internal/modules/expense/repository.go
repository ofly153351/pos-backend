package expense

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	ListCategories(ctx context.Context, storeID string) ([]ExpenseCategory, error)
	GetCategory(ctx context.Context, storeID, categoryID string) (ExpenseCategory, error)
	CategoryNameExists(ctx context.Context, storeID, name, excludeID string) (bool, error)
	CreateCategory(ctx context.Context, category ExpenseCategory) (ExpenseCategory, error)
	UpdateCategory(ctx context.Context, storeID, categoryID string, updates map[string]any) error
	SeedDefaultCategories(ctx context.Context, storeID string, names []string) error

	ListExpenses(ctx context.Context, storeID string, query ExpenseListQuery) ([]Expense, int64, error)
	GetExpense(ctx context.Context, storeID, expenseID string) (Expense, error)
	CreateExpense(ctx context.Context, payload map[string]any) error
	UpdateExpense(ctx context.Context, storeID, expenseID string, updates map[string]any) (int64, error)
	VoidExpense(ctx context.Context, storeID, expenseID, voidedBy string) (int64, error)

	MonthAggregate(ctx context.Context, storeID string, from, to time.Time) (float64, int64, error)
	CategoryTotals(ctx context.Context, storeID string, from, to time.Time) ([]CategoryTotal, error)
	MonthlyTotals(ctx context.Context, storeID string, from, to time.Time) ([]MonthTotal, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

// ── Categories ──────────────────────────────────────────────────────────────

func (r PostgresRepository) ListCategories(ctx context.Context, storeID string) ([]ExpenseCategory, error) {
	var items []ExpenseCategory
	err := r.db.WithContext(ctx).
		Model(&ExpenseCategory{}).
		Where("store_id = ?", storeID).
		Order("sort_order ASC, name ASC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetCategory(ctx context.Context, storeID, categoryID string) (ExpenseCategory, error) {
	var item ExpenseCategory
	err := r.db.WithContext(ctx).
		Model(&ExpenseCategory{}).
		Where("store_id = ? AND id = ?", storeID, categoryID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ExpenseCategory{}, ErrCategoryNotFound
		}
		return ExpenseCategory{}, err
	}
	return item, nil
}

func (r PostgresRepository) CategoryNameExists(ctx context.Context, storeID, name, excludeID string) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&ExpenseCategory{}).
		Where("store_id = ? AND lower(name) = lower(?)", storeID, strings.TrimSpace(name))
	if excludeID != "" {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r PostgresRepository) CreateCategory(ctx context.Context, category ExpenseCategory) (ExpenseCategory, error) {
	payload := map[string]any{
		"id":         category.ID,
		"store_id":   category.StoreID,
		"name":       category.Name,
		"is_active":  category.IsActive,
		"sort_order": category.SortOrder,
		"created_at": category.CreatedAt,
		"updated_at": category.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Table("expense_categories").Create(payload).Error; err != nil {
		return ExpenseCategory{}, err
	}
	category.UpdatedAt = category.CreatedAt
	return category, nil
}

func (r PostgresRepository) UpdateCategory(ctx context.Context, storeID, categoryID string, updates map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&ExpenseCategory{}).
		Where("store_id = ? AND id = ?", storeID, categoryID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (r PostgresRepository) SeedDefaultCategories(ctx context.Context, storeID string, names []string) error {
	now := time.Now().UTC()
	rows := make([]map[string]any, 0, len(names))
	for index, name := range names {
		rows = append(rows, map[string]any{
			"id":         newCategoryID(),
			"store_id":   storeID,
			"name":       name,
			"is_active":  true,
			"sort_order": index + 1,
			"created_at": now,
			"updated_at": now,
		})
	}
	return r.db.WithContext(ctx).Table("expense_categories").Create(rows).Error
}

// ── Expenses ────────────────────────────────────────────────────────────────

const expenseSelect = "expenses.*, ec.name AS category_name, u.full_name AS created_by_name"

func (r PostgresRepository) expenseBase(ctx context.Context, storeID string) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("expenses").
		Joins("LEFT JOIN expense_categories ec ON ec.id = expenses.category_id").
		Joins("LEFT JOIN users u ON u.id = expenses.created_by").
		Where("expenses.store_id = ? AND expenses.voided_at IS NULL", storeID)
}

func applyExpenseFilters(query *gorm.DB, q ExpenseListQuery) *gorm.DB {
	if q.From != "" {
		query = query.Where("expenses.expense_date >= ?", q.From)
	}
	if q.To != "" {
		query = query.Where("expenses.expense_date <= ?", q.To)
	}
	if q.CategoryID != "" {
		query = query.Where("expenses.category_id = ?", q.CategoryID)
	}
	if q.PaymentMethod != "" {
		query = query.Where("expenses.payment_method = ?", q.PaymentMethod)
	}
	return query
}

func (r PostgresRepository) ListExpenses(ctx context.Context, storeID string, q ExpenseListQuery) ([]Expense, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}

	var total int64
	countQuery := applyExpenseFilters(
		r.db.WithContext(ctx).
			Table("expenses").
			Where("expenses.store_id = ? AND expenses.voided_at IS NULL", storeID),
		q,
	)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []Expense
	err := applyExpenseFilters(r.expenseBase(ctx, storeID), q).
		Select(expenseSelect).
		Order("expenses.expense_date DESC, expenses.created_at DESC").
		Limit(q.Limit).
		Offset((q.Page - 1) * q.Limit).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r PostgresRepository) GetExpense(ctx context.Context, storeID, expenseID string) (Expense, error) {
	var item Expense
	err := r.expenseBase(ctx, storeID).
		Select(expenseSelect).
		Where("expenses.id = ?", expenseID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Expense{}, ErrExpenseNotFound
		}
		return Expense{}, err
	}
	return item, nil
}

func (r PostgresRepository) CreateExpense(ctx context.Context, payload map[string]any) error {
	return r.db.WithContext(ctx).Table("expenses").Create(payload).Error
}

func (r PostgresRepository) UpdateExpense(ctx context.Context, storeID, expenseID string, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Table("expenses").
		Where("store_id = ? AND id = ? AND voided_at IS NULL", storeID, expenseID).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r PostgresRepository) VoidExpense(ctx context.Context, storeID, expenseID, voidedBy string) (int64, error) {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Table("expenses").
		Where("store_id = ? AND id = ? AND voided_at IS NULL", storeID, expenseID).
		Updates(map[string]any{
			"voided_at":  now,
			"voided_by":  voidedBy,
			"updated_at": now,
		})
	return result.RowsAffected, result.Error
}

// ── Aggregates (summary) ────────────────────────────────────────────────────

func (r PostgresRepository) summaryBase(ctx context.Context, storeID string, from, to time.Time) *gorm.DB {
	// Columns are table-qualified because CategoryTotals joins expense_categories,
	// which also carries store_id/created_at — bare names become ambiguous.
	return r.db.WithContext(ctx).
		Table("expenses").
		Where("expenses.store_id = ? AND expenses.voided_at IS NULL AND expenses.status = ?", storeID, StatusApproved).
		Where("expenses.expense_date >= ? AND expenses.expense_date < ?", from.Format("2006-01-02"), to.Format("2006-01-02"))
}

func (r PostgresRepository) MonthAggregate(ctx context.Context, storeID string, from, to time.Time) (float64, int64, error) {
	var row struct {
		Total float64 `gorm:"column:total"`
		Count int64   `gorm:"column:count"`
	}
	err := r.summaryBase(ctx, storeID, from, to).
		Select("COALESCE(SUM(amount), 0) AS total, COUNT(*) AS count").
		Take(&row).Error
	if err != nil {
		return 0, 0, err
	}
	return row.Total, row.Count, nil
}

func (r PostgresRepository) CategoryTotals(ctx context.Context, storeID string, from, to time.Time) ([]CategoryTotal, error) {
	var rows []CategoryTotal
	err := r.summaryBase(ctx, storeID, from, to).
		Select("expenses.category_id AS category_id, COALESCE(ec.name, '') AS name, COALESCE(SUM(expenses.amount), 0) AS total").
		Joins("LEFT JOIN expense_categories ec ON ec.id = expenses.category_id").
		Group("expenses.category_id, ec.name").
		Order("total DESC").
		Find(&rows).Error
	return rows, err
}

func (r PostgresRepository) MonthlyTotals(ctx context.Context, storeID string, from, to time.Time) ([]MonthTotal, error) {
	var rows []MonthTotal
	err := r.summaryBase(ctx, storeID, from, to).
		Select("to_char(expense_date, 'YYYY-MM') AS month, COALESCE(SUM(amount), 0) AS total").
		Group("to_char(expense_date, 'YYYY-MM')").
		Order("month ASC").
		Find(&rows).Error
	return rows, err
}
