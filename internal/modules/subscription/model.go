package subscription

import "time"

type Plan struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	DurationDays int       `json:"duration_days"`
	PriceAmount  float64   `json:"price_amount"`
	CurrencyCode string    `json:"currency_code"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type StoreSubscription struct {
	ID                 string    `json:"id"`
	StoreID            string    `json:"store_id"`
	StoreName          string    `json:"store_name,omitempty"`
	OwnerUserID        string    `json:"owner_user_id,omitempty"`
	OwnerEmail         string    `json:"owner_email,omitempty"`
	PlanID             string    `json:"plan_id"`
	PlanCode           string    `json:"plan_code"`
	PlanName           string    `json:"plan_name"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CreatedAt          time.Time `json:"created_at"`
}

type ChangeSubscriptionRequest struct {
	PlanCode string `json:"plan_code"`
}

type UpdateSubscriptionStatusRequest struct {
	Status string `json:"status"`
}
