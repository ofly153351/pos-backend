package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Repository interface {
	CreateWithOwner(ctx context.Context, store Store, ownerUserID string, planCode string) (Store, error)
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) CreateWithOwner(ctx context.Context, storeModel Store, ownerUserID string, planCode string) (Store, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Store{}, err
	}
	defer tx.Rollback()

	var planID string
	var durationDays int
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, duration_days FROM subscription_plans WHERE code = $1 AND is_active = TRUE`,
		strings.TrimSpace(planCode),
	).Scan(&planID, &durationDays)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Store{}, ErrSubscriptionPlanNotFound
		}
		return Store{}, err
	}

	query := `
		INSERT INTO stores (id, owner_user_id, name, slug, logo_url, phone, address, currency_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`
	_, err = tx.ExecContext(
		ctx,
		query,
		storeModel.ID,
		ownerUserID,
		storeModel.Name,
		storeModel.Slug,
		storeModel.LogoURL,
		storeModel.Phone,
		storeModel.Address,
		storeModel.CurrencyCode,
		storeModel.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Store{}, ErrStoreSlugExists
		}
		return Store{}, err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO store_members (id, store_id, user_id, role) VALUES ($1, $2, $3, $4)`,
		newHexID(),
		storeModel.ID,
		ownerUserID,
		"owner",
	)
	if err != nil {
		return Store{}, err
	}

	periodEnd := storeModel.CreatedAt.Add(time.Duration(durationDays) * 24 * time.Hour)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO store_subscriptions (id, store_id, plan_id, status, current_period_start, current_period_end) VALUES ($1, $2, $3, $4, $5, $6)`,
		newHexID(),
		storeModel.ID,
		planID,
		"active",
		storeModel.CreatedAt,
		periodEnd,
	)
	if err != nil {
		return Store{}, err
	}

	if err := tx.Commit(); err != nil {
		return Store{}, err
	}

	storeModel.SubscriptionPlanCode = planCode
	storeModel.SubscriptionStatus = "active"
	storeModel.SubscriptionPeriodEnd = periodEnd
	return storeModel, nil
}

func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}

	query := `
		SELECT 1
		FROM store_members
		WHERE store_id = $1
		  AND user_id = $2
		  AND role IN ('owner', 'manager')
	`

	var ok int
	err := r.db.QueryRowContext(ctx, query, storeID, userID).Scan(&ok)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
