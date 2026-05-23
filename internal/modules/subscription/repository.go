package subscription

import (
	"context"
	"errors"
	"pos-backend/internal/idgen"
	"time"

	"gorm.io/gorm"
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
	db *gorm.DB
}

type storeSubscriptionView struct {
	ID                 string    `gorm:"column:id"`
	StoreID            string    `gorm:"column:store_id"`
	StoreName          string    `gorm:"column:store_name"`
	OwnerUserID        string    `gorm:"column:owner_user_id"`
	OwnerEmail         string    `gorm:"column:owner_email"`
	PlanID             string    `gorm:"column:plan_id"`
	PlanCode           string    `gorm:"column:plan_code"`
	PlanName           string    `gorm:"column:plan_name"`
	Status             string    `gorm:"column:status"`
	CurrentPeriodStart time.Time `gorm:"column:current_period_start"`
	CurrentPeriodEnd   time.Time `gorm:"column:current_period_end"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) ListPlans(ctx context.Context) ([]Plan, error) {
	var plans []Plan
	err := r.db.WithContext(ctx).
		Model(&Plan{}).
		Where("is_active = TRUE").
		Order("price_amount ASC").
		Find(&plans).Error
	return plans, err
}

func (r PostgresRepository) ListAll(ctx context.Context) ([]StoreSubscription, error) {
	var rows []storeSubscriptionView
	err := r.db.WithContext(ctx).
		Table("store_subscriptions ss").
		Select("ss.id, ss.store_id, s.name AS store_name, s.owner_user_id, COALESCE(u.email, '') AS owner_email, ss.plan_id, sp.code AS plan_code, sp.name AS plan_name, ss.status, ss.current_period_start, ss.current_period_end, ss.created_at").
		Joins("JOIN stores s ON s.id = ss.store_id").
		Joins("LEFT JOIN users u ON u.id = s.owner_user_id").
		Joins("JOIN subscription_plans sp ON sp.id = ss.plan_id").
		Order("ss.created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]StoreSubscription, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toSubscription())
	}
	return items, nil
}

func (r PostgresRepository) GetCurrentByStore(ctx context.Context, storeID string) (StoreSubscription, error) {
	var row storeSubscriptionView
	err := r.db.WithContext(ctx).
		Table("store_subscriptions ss").
		Select("ss.id, ss.store_id, s.name AS store_name, s.owner_user_id, COALESCE(u.email, '') AS owner_email, ss.plan_id, sp.code AS plan_code, sp.name AS plan_name, ss.status, ss.current_period_start, ss.current_period_end, ss.created_at").
		Joins("JOIN stores s ON s.id = ss.store_id").
		Joins("LEFT JOIN users u ON u.id = s.owner_user_id").
		Joins("JOIN subscription_plans sp ON sp.id = ss.plan_id").
		Where("ss.store_id = ?", storeID).
		Order("ss.created_at DESC").
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return StoreSubscription{}, ErrSubscriptionNotFound
		}
		return StoreSubscription{}, err
	}
	return row.toSubscription(), nil
}

func (r PostgresRepository) ChangePlan(ctx context.Context, storeID, planCode string, changedAt time.Time) (StoreSubscription, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return StoreSubscription{}, tx.Error
	}
	defer tx.Rollback()

	var plan Plan
	err := tx.
		Model(&Plan{}).
		Where("code = ? AND is_active = TRUE", planCode).
		Take(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return StoreSubscription{}, ErrPlanNotFound
		}
		return StoreSubscription{}, err
	}

	if err := tx.Model(&StoreSubscription{}).
		Where("store_id = ? AND status IN ?", storeID, []string{"trialing", "active", "past_due"}).
		Update("status", "cancelled").Error; err != nil {
		return StoreSubscription{}, err
	}

	sub := StoreSubscription{
		ID:                 idgen.Generate(idgen.PrefixStoreSubscription),
		StoreID:            storeID,
		PlanID:             plan.ID,
		PlanCode:           plan.Code,
		PlanName:           plan.Name,
		Status:             "active",
		CurrentPeriodStart: changedAt,
		CurrentPeriodEnd:   changedAt.Add(time.Duration(plan.DurationDays) * 24 * time.Hour),
		CreatedAt:          changedAt,
	}
	if err := tx.Omit("Plan").Create(&sub).Error; err != nil {
		return StoreSubscription{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return StoreSubscription{}, err
	}
	return r.GetCurrentByStore(ctx, storeID)
}

func (r PostgresRepository) UpdateStatus(ctx context.Context, storeID, status string) (StoreSubscription, error) {
	latestSubQuery := r.db.WithContext(ctx).
		Model(&StoreSubscription{}).
		Select("id").
		Where("store_id = ?", storeID).
		Order("created_at DESC").
		Limit(1)

	result := r.db.WithContext(ctx).
		Model(&StoreSubscription{}).
		Where("id = (?)", latestSubQuery).
		Update("status", status)
	if result.Error != nil {
		return StoreSubscription{}, result.Error
	}
	if result.RowsAffected == 0 {
		return StoreSubscription{}, ErrSubscriptionNotFound
	}
	return r.GetCurrentByStore(ctx, storeID)
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

func (row storeSubscriptionView) toSubscription() StoreSubscription {
	return StoreSubscription{
		ID:                 row.ID,
		StoreID:            row.StoreID,
		StoreName:          row.StoreName,
		OwnerUserID:        row.OwnerUserID,
		OwnerEmail:         row.OwnerEmail,
		PlanID:             row.PlanID,
		PlanCode:           row.PlanCode,
		PlanName:           row.PlanName,
		Status:             row.Status,
		CurrentPeriodStart: row.CurrentPeriodStart,
		CurrentPeriodEnd:   row.CurrentPeriodEnd,
		CreatedAt:          row.CreatedAt,
	}
}

