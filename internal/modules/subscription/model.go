package subscription

import "time"

type Plan struct {
	ID           string    `json:"id" gorm:"column:id;primaryKey"`
	Code         string    `json:"code" gorm:"column:code"`
	Name         string    `json:"name" gorm:"column:name"`
	Description  string    `json:"description,omitempty" gorm:"column:description"`
	DurationDays int       `json:"duration_days" gorm:"column:duration_days"`
	PriceAmount  float64   `json:"price_amount" gorm:"column:price_amount"`
	CurrencyCode string    `json:"currency_code" gorm:"column:currency_code"`
	IsActive     bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
}

type StoreSubscription struct {
	ID                 string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID            string    `json:"store_id" gorm:"column:store_id"`
	StoreName          string    `json:"store_name,omitempty" gorm:"-"`
	OwnerUserID        string    `json:"owner_user_id,omitempty" gorm:"-"`
	OwnerEmail         string    `json:"owner_email,omitempty" gorm:"-"`
	PlanID             string    `json:"plan_id" gorm:"column:plan_id"`
	PlanCode           string    `json:"plan_code" gorm:"-"`
	PlanName           string    `json:"plan_name" gorm:"-"`
	Status             string    `json:"status" gorm:"column:status"`
	CurrentPeriodStart time.Time `json:"current_period_start" gorm:"column:current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end" gorm:"column:current_period_end"`
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at"`
	Plan               Plan      `json:"-" gorm:"foreignKey:PlanID;references:ID"`
}

func (Plan) TableName() string { return "subscription_plans" }
func (StoreSubscription) TableName() string {
	return "store_subscriptions"
}

type ChangeSubscriptionRequest struct {
	PlanCode string `json:"plan_code"`
}

type UpdateSubscriptionStatusRequest struct {
	Status string `json:"status"`
}
