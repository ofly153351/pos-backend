package subscription

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

type Repository interface {
	ListPlans(ctx context.Context) ([]Plan, error)
	ListAll(ctx context.Context) ([]StoreSubscription, error)
	GetCurrentByStore(ctx context.Context, storeID string) (StoreSubscription, error)
	ChangePlan(ctx context.Context, storeID, planCode string, changedAt time.Time) (StoreSubscription, error)
	UpdateStatus(ctx context.Context, storeID, status string) (StoreSubscription, error)
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) ListPlans(ctx context.Context) ([]Plan, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, code, name, COALESCE(description, ''), duration_days, price_amount, currency_code, is_active, created_at
		 FROM subscription_plans
		 WHERE is_active = TRUE
		 ORDER BY price_amount ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var plan Plan
		if err := rows.Scan(
			&plan.ID,
			&plan.Code,
			&plan.Name,
			&plan.Description,
			&plan.DurationDays,
			&plan.PriceAmount,
			&plan.CurrencyCode,
			&plan.IsActive,
			&plan.CreatedAt,
		); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}

	return plans, rows.Err()
}

func (r PostgresRepository) ListAll(ctx context.Context) ([]StoreSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ss.id, ss.store_id, s.name, s.owner_user_id, COALESCE(u.email, ''), ss.plan_id, sp.code, sp.name, ss.status, ss.current_period_start, ss.current_period_end, ss.created_at
		FROM store_subscriptions ss
		JOIN stores s ON s.id = ss.store_id
		LEFT JOIN users u ON u.id = s.owner_user_id
		JOIN subscription_plans sp ON sp.id = ss.plan_id
		ORDER BY ss.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []StoreSubscription
	for rows.Next() {
		var sub StoreSubscription
		if err := rows.Scan(
			&sub.ID,
			&sub.StoreID,
			&sub.StoreName,
			&sub.OwnerUserID,
			&sub.OwnerEmail,
			&sub.PlanID,
			&sub.PlanCode,
			&sub.PlanName,
			&sub.Status,
			&sub.CurrentPeriodStart,
			&sub.CurrentPeriodEnd,
			&sub.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, sub)
	}

	return items, rows.Err()
}

func (r PostgresRepository) GetCurrentByStore(ctx context.Context, storeID string) (StoreSubscription, error) {
	query := `
		SELECT ss.id, ss.store_id, s.name, s.owner_user_id, COALESCE(u.email, ''), ss.plan_id, sp.code, sp.name, ss.status, ss.current_period_start, ss.current_period_end, ss.created_at
		FROM store_subscriptions ss
		JOIN stores s ON s.id = ss.store_id
		LEFT JOIN users u ON u.id = s.owner_user_id
		JOIN subscription_plans sp ON sp.id = ss.plan_id
		WHERE ss.store_id = $1
		ORDER BY ss.created_at DESC
		LIMIT 1
	`

	var sub StoreSubscription
	err := r.db.QueryRowContext(ctx, query, storeID).Scan(
		&sub.ID,
		&sub.StoreID,
		&sub.StoreName,
		&sub.OwnerUserID,
		&sub.OwnerEmail,
		&sub.PlanID,
		&sub.PlanCode,
		&sub.PlanName,
		&sub.Status,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StoreSubscription{}, ErrSubscriptionNotFound
		}
		return StoreSubscription{}, err
	}

	return sub, nil
}

func (r PostgresRepository) ChangePlan(ctx context.Context, storeID, planCode string, changedAt time.Time) (StoreSubscription, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StoreSubscription{}, err
	}
	defer tx.Rollback()

	var plan Plan
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, code, name, COALESCE(description, ''), duration_days, price_amount, currency_code, is_active, created_at
		 FROM subscription_plans WHERE code = $1 AND is_active = TRUE`,
		planCode,
	).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.Description,
		&plan.DurationDays,
		&plan.PriceAmount,
		&plan.CurrencyCode,
		&plan.IsActive,
		&plan.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StoreSubscription{}, ErrPlanNotFound
		}
		return StoreSubscription{}, err
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE store_subscriptions
		 SET status = 'cancelled'
		 WHERE store_id = $1 AND status IN ('trialing', 'active', 'past_due')`,
		storeID,
	)
	if err != nil {
		return StoreSubscription{}, err
	}

	sub := StoreSubscription{
		ID:                 productNewID(),
		StoreID:            storeID,
		PlanID:             plan.ID,
		PlanCode:           plan.Code,
		PlanName:           plan.Name,
		Status:             "active",
		CurrentPeriodStart: changedAt,
		CurrentPeriodEnd:   changedAt.Add(time.Duration(plan.DurationDays) * 24 * time.Hour),
		CreatedAt:          changedAt,
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO store_subscriptions (id, store_id, plan_id, status, current_period_start, current_period_end, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sub.ID,
		sub.StoreID,
		sub.PlanID,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.CreatedAt,
	)
	if err != nil {
		return StoreSubscription{}, err
	}

	if err := tx.Commit(); err != nil {
		return StoreSubscription{}, err
	}

	return r.GetCurrentByStore(ctx, storeID)
}

func (r PostgresRepository) UpdateStatus(ctx context.Context, storeID, status string) (StoreSubscription, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE store_subscriptions
		SET status = $2
		WHERE id = (
			SELECT id FROM store_subscriptions
			WHERE store_id = $1
			ORDER BY created_at DESC
			LIMIT 1
		)
	`, storeID, status)
	if err != nil {
		return StoreSubscription{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return StoreSubscription{}, err
	}
	if affected == 0 {
		return StoreSubscription{}, ErrSubscriptionNotFound
	}

	return r.GetCurrentByStore(ctx, storeID)
}

func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}

	var exists int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM store_members WHERE store_id = $1 AND user_id = $2 AND role IN ('owner', 'manager')`,
		storeID,
		userID,
	).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func productNewID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "generated-id"
	}

	return hex.EncodeToString(buf)
}
