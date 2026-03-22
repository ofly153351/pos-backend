package store

import (
	"mime/multipart"
	"time"
)

type Store struct {
	ID                    string    `json:"id"`
	OwnerUserID           string    `json:"owner_user_id"`
	Name                  string    `json:"name"`
	Slug                  string    `json:"slug"`
	LogoURL               string    `json:"logo_url,omitempty"`
	Phone                 string    `json:"phone,omitempty"`
	Address               string    `json:"address,omitempty"`
	CurrencyCode          string    `json:"currency_code"`
	SubscriptionPlanCode  string    `json:"subscription_plan_code"`
	SubscriptionStatus    string    `json:"subscription_status"`
	SubscriptionPeriodEnd time.Time `json:"subscription_period_end"`
	CreatedAt             time.Time `json:"created_at"`
}

type CreateStoreRequest struct {
	Name                 string
	Slug                 string
	Phone                string
	Address              string
	CurrencyCode         string
	SubscriptionPlanCode string
	LogoFile             *multipart.FileHeader
}
