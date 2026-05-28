package store

import (
	"mime/multipart"
	"time"
)

type Store struct {
	ID                    string    `json:"id" gorm:"column:id;primaryKey"`
	OwnerUserID           string    `json:"owner_user_id" gorm:"column:owner_user_id"`
	Name                  string    `json:"name" gorm:"column:name"`
	LogoURL               string    `json:"logo_url,omitempty" gorm:"column:logo_url"`
	Phone                 string    `json:"phone,omitempty" gorm:"column:phone"`
	Address               string    `json:"address,omitempty" gorm:"column:address"`
	PromptPayID           string    `json:"promptpay_id,omitempty" gorm:"column:promptpay_id"`
	TaxID                 string    `json:"tax_id,omitempty" gorm:"column:tax_id"`
	CurrencyCode          string    `json:"currency_code" gorm:"column:currency_code"`
	SubscriptionPlanCode  string    `json:"subscription_plan_code" gorm:"-"`
	SubscriptionStatus    string    `json:"subscription_status" gorm:"-"`
	SubscriptionPeriodEnd time.Time `json:"subscription_period_end" gorm:"-"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
}

func (Store) TableName() string {
	return "stores"
}

type CreateStoreRequest struct {
	Name                 string
	Phone                string
	Address              string
	PromptPayID          string
	CurrencyCode         string
	SubscriptionPlanCode string
	LogoFile             *multipart.FileHeader
}

type UpdateStoreRequest struct {
	Name         *string `json:"name"`
	Phone        *string `json:"phone"`
	Address      *string `json:"address"`
	PromptPayID  *string `json:"promptpay_id"`
	TaxID        *string `json:"tax_id"`
	CurrencyCode *string `json:"currency_code"`
	LogoFile     *multipart.FileHeader
}
