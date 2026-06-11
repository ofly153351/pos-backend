package expense

import (
	"context"
	"math"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

// ── Access rules ────────────────────────────────────────────────────────────
// Phase 2a policy:
//   - every store member (owner/manager/cashier) can view and create
//   - owner + manager can edit expenses and manage categories
//   - owner only can delete (soft-void) expenses
// Enforced here in the service layer — the UI hiding buttons is cosmetic only.

func (s Service) ensureStoreAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrExpenseStoreIDRequired
	}
	ok, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !ok {
		return ErrExpenseForbidden
	}
	return nil
}

func (s Service) ensureManageAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if actor.Role == auth.RolePlatformAdmin {
		return nil
	}
	if actor.Role != auth.RoleOwner && actor.Role != auth.RoleManager {
		return ErrExpenseEditForbidden
	}
	return s.ensureStoreAccess(ctx, actor, storeID)
}

func (s Service) ensureDeleteAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if actor.Role == auth.RolePlatformAdmin {
		return nil
	}
	if actor.Role != auth.RoleOwner {
		return ErrExpenseDeleteForbidden
	}
	return s.ensureStoreAccess(ctx, actor, storeID)
}

// ── Categories ──────────────────────────────────────────────────────────────

func (s Service) ListCategories(ctx context.Context, actor auth.Claims, storeID string) ([]ExpenseCategory, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListCategories(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		// Lazy-seed defaults for stores created before migration 020 (or new stores).
		if err := s.repo.SeedDefaultCategories(ctx, storeID, DefaultCategoryNames); err != nil {
			return nil, err
		}
		return s.repo.ListCategories(ctx, storeID)
	}
	return items, nil
}

func (s Service) CreateCategory(ctx context.Context, actor auth.Claims, storeID string, input CreateCategoryRequest) (ExpenseCategory, error) {
	if err := s.ensureManageAccess(ctx, actor, storeID); err != nil {
		return ExpenseCategory{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return ExpenseCategory{}, ErrCategoryNameRequired
	}
	exists, err := s.repo.CategoryNameExists(ctx, storeID, name, "")
	if err != nil {
		return ExpenseCategory{}, err
	}
	if exists {
		return ExpenseCategory{}, ErrCategoryDuplicate
	}

	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}

	item := ExpenseCategory{
		ID:        newCategoryID(),
		StoreID:   storeID,
		Name:      name,
		IsActive:  true,
		SortOrder: sortOrder,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.CreateCategory(ctx, item)
}

func (s Service) UpdateCategory(ctx context.Context, actor auth.Claims, storeID, categoryID string, input UpdateCategoryRequest) (ExpenseCategory, error) {
	if err := s.ensureManageAccess(ctx, actor, storeID); err != nil {
		return ExpenseCategory{}, err
	}
	if _, err := s.repo.GetCategory(ctx, storeID, categoryID); err != nil {
		return ExpenseCategory{}, err
	}

	updates := map[string]any{"updated_at": time.Now().UTC()}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return ExpenseCategory{}, ErrCategoryNameRequired
		}
		exists, err := s.repo.CategoryNameExists(ctx, storeID, name, categoryID)
		if err != nil {
			return ExpenseCategory{}, err
		}
		if exists {
			return ExpenseCategory{}, ErrCategoryDuplicate
		}
		updates["name"] = name
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}

	if err := s.repo.UpdateCategory(ctx, storeID, categoryID, updates); err != nil {
		return ExpenseCategory{}, err
	}
	return s.repo.GetCategory(ctx, storeID, categoryID)
}

// DeactivateCategory soft-disables a category (expenses keep referencing it).
func (s Service) DeactivateCategory(ctx context.Context, actor auth.Claims, storeID, categoryID string) error {
	if err := s.ensureManageAccess(ctx, actor, storeID); err != nil {
		return err
	}
	return s.repo.UpdateCategory(ctx, storeID, categoryID, map[string]any{
		"is_active":  false,
		"updated_at": time.Now().UTC(),
	})
}

// ── Expenses ────────────────────────────────────────────────────────────────

func parseExpenseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, ErrInvalidExpenseDate
	}
	return parsed, nil
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func (s Service) validateCategory(ctx context.Context, storeID, categoryID string) error {
	category, err := s.repo.GetCategory(ctx, storeID, strings.TrimSpace(categoryID))
	if err != nil {
		return ErrInvalidCategory
	}
	if !category.IsActive {
		return ErrInvalidCategory
	}
	return nil
}

func (s Service) ListExpenses(ctx context.Context, actor auth.Claims, storeID string, query ExpenseListQuery) (ExpenseListResult, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return ExpenseListResult{}, err
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 200 {
		query.Limit = 20
	}

	items, total, err := s.repo.ListExpenses(ctx, storeID, query)
	if err != nil {
		return ExpenseListResult{}, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))
	if totalPages < 1 {
		totalPages = 1
	}
	return ExpenseListResult{
		Items:      items,
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s Service) GetExpense(ctx context.Context, actor auth.Claims, storeID, expenseID string) (Expense, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return Expense{}, err
	}
	return s.repo.GetExpense(ctx, storeID, expenseID)
}

func (s Service) CreateExpense(ctx context.Context, actor auth.Claims, storeID string, input CreateExpenseRequest) (Expense, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return Expense{}, err
	}

	expenseDate, err := parseExpenseDate(input.ExpenseDate)
	if err != nil {
		return Expense{}, err
	}
	if strings.TrimSpace(input.Description) == "" {
		return Expense{}, ErrInvalidDescription
	}
	amount := roundMoney(input.Amount)
	if amount <= 0 {
		return Expense{}, ErrInvalidAmount
	}
	method := strings.ToLower(strings.TrimSpace(input.PaymentMethod))
	if method == "" {
		method = "cash"
	}
	if !AllowedPaymentMethods[method] {
		return Expense{}, ErrInvalidPaymentMethod
	}
	if err := s.validateCategory(ctx, storeID, input.CategoryID); err != nil {
		return Expense{}, err
	}

	now := time.Now().UTC()
	id := newExpenseID()
	payload := map[string]any{
		"id":             id,
		"store_id":       storeID,
		"expense_date":   expenseDate.Format("2006-01-02"),
		"category_id":    strings.TrimSpace(input.CategoryID),
		"description":    strings.TrimSpace(input.Description),
		"amount":         amount,
		"payment_method": method,
		"status":         StatusApproved, // Phase 2a: every expense lands approved
		"created_by":     actor.UserID,
		"created_at":     now,
		"updated_at":     now,
	}
	if strings.TrimSpace(input.Note) == "" {
		payload["note"] = nil
	} else {
		payload["note"] = strings.TrimSpace(input.Note)
	}

	if err := s.repo.CreateExpense(ctx, payload); err != nil {
		return Expense{}, err
	}
	return s.repo.GetExpense(ctx, storeID, id)
}

func (s Service) UpdateExpense(ctx context.Context, actor auth.Claims, storeID, expenseID string, input UpdateExpenseRequest) (Expense, error) {
	if err := s.ensureManageAccess(ctx, actor, storeID); err != nil {
		return Expense{}, err
	}
	if _, err := s.repo.GetExpense(ctx, storeID, expenseID); err != nil {
		return Expense{}, err
	}

	updates := map[string]any{"updated_at": time.Now().UTC()}
	if input.ExpenseDate != nil {
		expenseDate, err := parseExpenseDate(*input.ExpenseDate)
		if err != nil {
			return Expense{}, err
		}
		updates["expense_date"] = expenseDate.Format("2006-01-02")
	}
	if input.CategoryID != nil {
		if err := s.validateCategory(ctx, storeID, *input.CategoryID); err != nil {
			return Expense{}, err
		}
		updates["category_id"] = strings.TrimSpace(*input.CategoryID)
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if description == "" {
			return Expense{}, ErrInvalidDescription
		}
		updates["description"] = description
	}
	if input.Amount != nil {
		amount := roundMoney(*input.Amount)
		if amount <= 0 {
			return Expense{}, ErrInvalidAmount
		}
		updates["amount"] = amount
	}
	if input.PaymentMethod != nil {
		method := strings.ToLower(strings.TrimSpace(*input.PaymentMethod))
		if !AllowedPaymentMethods[method] {
			return Expense{}, ErrInvalidPaymentMethod
		}
		updates["payment_method"] = method
	}
	if input.Note != nil {
		note := strings.TrimSpace(*input.Note)
		if note == "" {
			updates["note"] = nil
		} else {
			updates["note"] = note
		}
	}

	affected, err := s.repo.UpdateExpense(ctx, storeID, expenseID, updates)
	if err != nil {
		return Expense{}, err
	}
	if affected == 0 {
		return Expense{}, ErrExpenseNotFound
	}
	return s.repo.GetExpense(ctx, storeID, expenseID)
}

// VoidExpense soft-deletes (audit-safe) — owner only.
func (s Service) VoidExpense(ctx context.Context, actor auth.Claims, storeID, expenseID string) error {
	if err := s.ensureDeleteAccess(ctx, actor, storeID); err != nil {
		return err
	}
	affected, err := s.repo.VoidExpense(ctx, storeID, expenseID, actor.UserID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrExpenseNotFound
	}
	return nil
}

// ── Summary ─────────────────────────────────────────────────────────────────

// Summary aggregates the current month KPIs + a 6-month trend, server-side.
func (s Service) Summary(ctx context.Context, actor auth.Claims, storeID string) (ExpenseSummary, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return ExpenseSummary{}, err
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)
	trendStart := monthStart.AddDate(0, -5, 0)

	total, count, err := s.repo.MonthAggregate(ctx, storeID, monthStart, monthEnd)
	if err != nil {
		return ExpenseSummary{}, err
	}
	byCategory, err := s.repo.CategoryTotals(ctx, storeID, monthStart, monthEnd)
	if err != nil {
		return ExpenseSummary{}, err
	}
	monthly, err := s.repo.MonthlyTotals(ctx, storeID, trendStart, monthEnd)
	if err != nil {
		return ExpenseSummary{}, err
	}

	// Fill all 6 buckets so the chart never has missing months.
	totalsByMonth := make(map[string]float64, len(monthly))
	for _, row := range monthly {
		totalsByMonth[row.Month] = row.Total
	}
	trend := make([]MonthTotal, 0, 6)
	for i := 0; i < 6; i++ {
		bucket := trendStart.AddDate(0, i, 0).Format("2006-01")
		trend = append(trend, MonthTotal{Month: bucket, Total: totalsByMonth[bucket]})
	}

	summary := ExpenseSummary{
		MonthlyTotal: total,
		MonthlyCount: count,
		ByCategory:   byCategory,
		Trend:        trend,
	}
	if count > 0 {
		summary.AverageAmount = roundMoney(total / float64(count))
	}
	if len(byCategory) > 0 {
		summary.TopCategoryName = byCategory[0].Name
		summary.TopCategoryTotal = byCategory[0].Total
	}
	return summary, nil
}
