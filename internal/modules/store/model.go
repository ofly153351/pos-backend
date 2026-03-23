package store

import (
	"mime/multipart"
	"time"
)

type Store struct {
	ID                    string    `json:"id" gorm:"column:id;primaryKey"`
	OwnerUserID           string    `json:"owner_user_id" gorm:"column:owner_user_id"`
	Name                  string    `json:"name" gorm:"column:name"`
	Slug                  string    `json:"slug" gorm:"column:slug"`
	LogoURL               string    `json:"logo_url,omitempty" gorm:"column:logo_url"`
	Phone                 string    `json:"phone,omitempty" gorm:"column:phone"`
	Address               string    `json:"address,omitempty" gorm:"column:address"`
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
	Slug                 string
	Phone                string
	Address              string
	CurrencyCode         string
	SubscriptionPlanCode string
	LogoFile             *multipart.FileHeader
}
