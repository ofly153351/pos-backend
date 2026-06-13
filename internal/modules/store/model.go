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
	Fax                   string    `json:"fax,omitempty" gorm:"column:fax"`
	Email                 string    `json:"email,omitempty" gorm:"column:email"`
	Website               string    `json:"website,omitempty" gorm:"column:website"`
	Address               string    `json:"address,omitempty" gorm:"column:address"`
	PromptPayID           string    `json:"promptpay_id,omitempty" gorm:"column:promptpay_id"`
	TaxID                 string    `json:"tax_id,omitempty" gorm:"column:tax_id"`
	CurrencyCode          string    `json:"currency_code" gorm:"column:currency_code"`
	// Role is the caller's store_members.role for THIS store (owner/manager/
	// cashier/warehouse), surfaced on /me/stores so the frontend can gate UI on
	// the real per-store role instead of the global users.role. Not a stored
	// column (gorm:"-") — it is projected per-user in ListByUser.
	Role                  string    `json:"role,omitempty" gorm:"-"`
	SubscriptionPlanCode  string    `json:"subscription_plan_code" gorm:"-"`
	SubscriptionStatus    string    `json:"subscription_status" gorm:"-"`
	SubscriptionPeriodEnd time.Time `json:"subscription_period_end" gorm:"-"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
}

func (Store) TableName() string {
	return "stores"
}

type StoreBankAccount struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(30)"`
	StoreID     string    `json:"store_id" gorm:"not null;index"`
	BankCode    string    `json:"bank_code" gorm:"not null;default:''"`
	BankName    string    `json:"bank_name" gorm:"not null;default:''"`
	AccountNo   string    `json:"account_no" gorm:"not null;default:''"`
	AccountName string    `json:"account_name" gorm:"not null;default:''"`
	CreatedAt   time.Time `json:"created_at"`
}

func (StoreBankAccount) TableName() string { return "store_bank_accounts" }

type CreateBankAccountRequest struct {
	BankCode    string `json:"bank_code"`
	BankName    string `json:"bank_name"`
	AccountNo   string `json:"account_no"`
	AccountName string `json:"account_name"`
}

type CreateStoreRequest struct {
	Name                 string
	Phone                string
	Fax                  string
	Email                string
	Website              string
	Address              string
	PromptPayID          string
	CurrencyCode         string
	SubscriptionPlanCode string
	LogoFile             *multipart.FileHeader
}

type UpdateStoreRequest struct {
	Name         *string `json:"name"`
	Phone        *string `json:"phone"`
	Fax          *string `json:"fax"`
	Email        *string `json:"email"`
	Website      *string `json:"website"`
	Address      *string `json:"address"`
	PromptPayID  *string `json:"promptpay_id"`
	TaxID        *string `json:"tax_id"`
	CurrencyCode *string `json:"currency_code"`
	LogoFile     *multipart.FileHeader
}
