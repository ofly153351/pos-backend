package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CreateWithOwner(ctx context.Context, store Store, ownerUserID string, planCode string) (Store, error)
	GetByID(ctx context.Context, storeID string) (Store, error)
	ListByUser(ctx context.Context, userID, role string) ([]Store, error)
	Update(ctx context.Context, storeID string, update Store) error
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) CreateWithOwner(ctx context.Context, storeModel Store, ownerUserID string, planCode string) (Store, error) {
	hasPromptPayColumn, err := r.hasStorePromptPayIDColumn(ctx)
	if err != nil {
		return Store{}, err
	}

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
	err = tx.
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
	if hasPromptPayColumn {
		if storeModel.PromptPayID == "" {
			storePayload["promptpay_id"] = nil
		} else {
			storePayload["promptpay_id"] = storeModel.PromptPayID
		}
	}
	if err := tx.Table("stores").Create(storePayload).Error; err != nil {
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
		ID:      newStoreMemberID(),
		StoreID: storeModel.ID,
		UserID:  ownerUserID,
		Role:    "owner",
	}).Error; err != nil {
		return Store{}, err
	}

	periodEnd := storeModel.CreatedAt.Add(time.Duration(plan.DurationDays) * 24 * time.Hour)
	if err := tx.Table("store_subscriptions").Create(&storeSubscription{
		ID:                 newStoreSubscriptionID(),
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
	hasPromptPayColumn, err := r.hasStorePromptPayIDColumn(ctx)
	if err != nil {
		return Store{}, err
	}

	var storeModel Store
	selectCols := `
		id, owner_user_id, name,
		COALESCE(logo_url, '') AS logo_url,
		COALESCE(phone, '') AS phone,
		COALESCE(fax, '') AS fax,
		COALESCE(email, '') AS email,
		COALESCE(website, '') AS website,
		COALESCE(address, '') AS address,
		COALESCE(tax_id, '') AS tax_id,
		currency_code, created_at
	`
	if hasPromptPayColumn {
		selectCols = `
			id, owner_user_id, name,
			COALESCE(logo_url, '') AS logo_url,
			COALESCE(phone, '') AS phone,
			COALESCE(fax, '') AS fax,
			COALESCE(email, '') AS email,
			COALESCE(website, '') AS website,
			COALESCE(address, '') AS address,
			COALESCE(promptpay_id, '') AS promptpay_id,
			COALESCE(tax_id, '') AS tax_id,
			currency_code, created_at
		`
	}
	query := r.db.WithContext(ctx).Table("stores").Select(selectCols)

	err = query.
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

func (r PostgresRepository) ListByUser(ctx context.Context, userID, role string) ([]Store, error) {
	hasPromptPayColumn, err := r.hasStorePromptPayIDColumn(ctx)
	if err != nil {
		return nil, err
	}

	type storeListRow struct {
		ID                    string     `gorm:"column:id"`
		OwnerUserID           string     `gorm:"column:owner_user_id"`
		Name                  string     `gorm:"column:name"`
		LogoURL               string     `gorm:"column:logo_url"`
		Phone                 string     `gorm:"column:phone"`
		Fax                   string     `gorm:"column:fax"`
		Email                 string     `gorm:"column:email"`
		Website               string     `gorm:"column:website"`
		Address               string     `gorm:"column:address"`
		PromptPayID           string     `gorm:"column:promptpay_id"`
		TaxID                 string     `gorm:"column:tax_id"`
		CurrencyCode          string     `gorm:"column:currency_code"`
		SubscriptionPlanCode  string     `gorm:"column:subscription_plan_code"`
		SubscriptionStatus    string     `gorm:"column:subscription_status"`
		SubscriptionPeriodEnd *time.Time `gorm:"column:subscription_period_end"`
		CreatedAt             time.Time  `gorm:"column:created_at"`
	}

	promptPaySelect := "'' AS promptpay_id"
	if hasPromptPayColumn {
		promptPaySelect = "COALESCE(s.promptpay_id, '') AS promptpay_id"
	}

	query := r.db.WithContext(ctx).
		Table("stores s").
		Select(fmt.Sprintf(`
			s.id,
			s.owner_user_id,
			s.name,
			COALESCE(s.logo_url, '') AS logo_url,
			COALESCE(s.phone, '') AS phone,
			COALESCE(s.fax, '') AS fax,
			COALESCE(s.email, '') AS email,
			COALESCE(s.website, '') AS website,
			COALESCE(s.address, '') AS address,
			%s,
			COALESCE(s.tax_id, '') AS tax_id,
			s.currency_code,
			COALESCE(sub.plan_code, '') AS subscription_plan_code,
			COALESCE(sub.status, '') AS subscription_status,
			sub.current_period_end AS subscription_period_end,
			s.created_at
		`, promptPaySelect)).
		Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (ss.store_id)
					ss.store_id,
					sp.code AS plan_code,
					ss.status,
					ss.current_period_end
				FROM store_subscriptions ss
				JOIN subscription_plans sp ON sp.id = ss.plan_id
				ORDER BY ss.store_id, ss.created_at DESC
			) sub ON sub.store_id = s.id
		`)

	if role != "platform_admin" {
		query = query.
			Joins("JOIN store_members sm ON sm.store_id = s.id").
			Where("sm.user_id = ?", userID)
	}

	var rows []storeListRow
	if err := query.Order("s.created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]Store, 0, len(rows))
	for _, row := range rows {
		item := Store{
			ID:                   row.ID,
			OwnerUserID:          row.OwnerUserID,
			Name:                 row.Name,
			LogoURL:              row.LogoURL,
			Phone:                row.Phone,
			Fax:                  row.Fax,
			Email:                row.Email,
			Website:              row.Website,
			Address:              row.Address,
			PromptPayID:          row.PromptPayID,
			TaxID:                row.TaxID,
			CurrencyCode:         row.CurrencyCode,
			SubscriptionPlanCode: row.SubscriptionPlanCode,
			SubscriptionStatus:   row.SubscriptionStatus,
			CreatedAt:            row.CreatedAt,
		}
		if row.SubscriptionPeriodEnd != nil {
			item.SubscriptionPeriodEnd = *row.SubscriptionPeriodEnd
		}
		result = append(result, item)
	}
	return result, nil
}

func (r PostgresRepository) Update(ctx context.Context, storeID string, update Store) error {
	hasPromptPayColumn, err := r.hasStorePromptPayIDColumn(ctx)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"name":          update.Name,
		"phone":         nilIfEmpty(update.Phone),
		"fax":           update.Fax,
		"email":         update.Email,
		"website":       update.Website,
		"address":       nilIfEmpty(update.Address),
		"tax_id":        update.TaxID,
		"currency_code": update.CurrencyCode,
		"updated_at":    time.Now().UTC(),
	}
	if hasPromptPayColumn {
		payload["promptpay_id"] = nilIfEmpty(update.PromptPayID)
	}
	if update.LogoURL == "" {
		payload["logo_url"] = nil
	} else {
		payload["logo_url"] = update.LogoURL
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Table("stores").
			Where("id = ?", storeID).
			Updates(payload)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrStoreNotFound
		}
		return nil
	})
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

func (r PostgresRepository) hasStorePromptPayIDColumn(ctx context.Context) (bool, error) {
	type columnLookup struct {
		Exists bool `gorm:"column:exists"`
	}

	var lookup columnLookup
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = current_schema()
					AND table_name = 'stores'
					AND column_name = 'promptpay_id'
			) AS exists
		`).
		Scan(&lookup).Error
	if err != nil {
		return false, err
	}
	return lookup.Exists, nil
}

func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
