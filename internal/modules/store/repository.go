package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	CreateWithOwner(ctx context.Context, store Store, ownerUserID string, planCode string) (Store, error)
	GetByID(ctx context.Context, storeID string) (Store, error)
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) CreateWithOwner(ctx context.Context, storeModel Store, ownerUserID string, planCode string) (Store, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return Store{}, tx.Error
	}
	defer tx.Rollback()

	type planRecord struct {
		ID           string `gorm:"column:id"`
		DurationDays int    `gorm:"column:duration_days"`
	}

	var plan planRecord
	err := tx.
		Table("subscription_plans").
		Select("id", "duration_days").
		Where("code = ? AND is_active = TRUE", strings.TrimSpace(planCode)).
		Take(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Store{}, ErrSubscriptionPlanNotFound
		}
		return Store{}, err
	}

	storePayload := map[string]any{
		"id":            storeModel.ID,
		"owner_user_id": ownerUserID,
		"name":          storeModel.Name,
		"slug":          storeModel.Slug,
		"currency_code": storeModel.CurrencyCode,
		"created_at":    storeModel.CreatedAt,
		"updated_at":    storeModel.CreatedAt,
	}
	if storeModel.LogoURL == "" {
		storePayload["logo_url"] = nil
	} else {
		storePayload["logo_url"] = storeModel.LogoURL
	}
	if storeModel.Phone == "" {
		storePayload["phone"] = nil
	} else {
		storePayload["phone"] = storeModel.Phone
	}
	if storeModel.Address == "" {
		storePayload["address"] = nil
	} else {
		storePayload["address"] = storeModel.Address
	}
	if err := tx.Table("stores").Create(storePayload).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Store{}, ErrStoreSlugExists
		}
		return Store{}, err
	}

	type storeMember struct {
		ID      string `gorm:"column:id;primaryKey"`
		StoreID string `gorm:"column:store_id"`
		UserID  string `gorm:"column:user_id"`
		Role    string `gorm:"column:role"`
	}
	type storeSubscription struct {
		ID                 string    `gorm:"column:id;primaryKey"`
		StoreID            string    `gorm:"column:store_id"`
		PlanID             string    `gorm:"column:plan_id"`
		Status             string    `gorm:"column:status"`
		CurrentPeriodStart time.Time `gorm:"column:current_period_start"`
		CurrentPeriodEnd   time.Time `gorm:"column:current_period_end"`
	}

	if err := tx.Table("store_members").Create(&storeMember{
		ID:      newHexID(),
		StoreID: storeModel.ID,
		UserID:  ownerUserID,
		Role:    "owner",
	}).Error; err != nil {
		return Store{}, err
	}

	periodEnd := storeModel.CreatedAt.Add(time.Duration(plan.DurationDays) * 24 * time.Hour)
	if err := tx.Table("store_subscriptions").Create(&storeSubscription{
		ID:                 newHexID(),
		StoreID:            storeModel.ID,
		PlanID:             plan.ID,
		Status:             "active",
		CurrentPeriodStart: storeModel.CreatedAt,
		CurrentPeriodEnd:   periodEnd,
	}).Error; err != nil {
		return Store{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Store{}, err
	}

	storeModel.SubscriptionPlanCode = planCode
	storeModel.SubscriptionStatus = "active"
	storeModel.SubscriptionPeriodEnd = periodEnd
	return storeModel, nil
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID string) (Store, error) {
	var storeModel Store
	err := r.db.WithContext(ctx).
		Model(&Store{}).
		Where("id = ?", storeID).
		Take(&storeModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Store{}, ErrStoreNotFound
		}
		return Store{}, err
	}

	type subscriptionView struct {
		PlanCode         string    `gorm:"column:plan_code"`
		Status           string    `gorm:"column:status"`
		CurrentPeriodEnd time.Time `gorm:"column:current_period_end"`
	}
	var sub subscriptionView
	err = r.db.WithContext(ctx).
		Table("store_subscriptions ss").
		Select("sp.code AS plan_code, ss.status, ss.current_period_end").
		Joins("JOIN subscription_plans sp ON sp.id = ss.plan_id").
		Where("ss.store_id = ?", storeID).
		Order("ss.created_at DESC").
		Limit(1).
		Take(&sub).Error
	if err == nil {
		storeModel.SubscriptionPlanCode = sub.PlanCode
		storeModel.SubscriptionStatus = sub.Status
		storeModel.SubscriptionPeriodEnd = sub.CurrentPeriodEnd
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Store{}, err
	}

	return storeModel, nil
}

func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}

	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
