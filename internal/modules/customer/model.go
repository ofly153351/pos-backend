package customer

import "time"

type Customer struct {
	ID        string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID   string    `json:"store_id" gorm:"column:store_id"`
	Level     int       `json:"level" gorm:"column:customer_level"`
	FullName  string    `json:"full_name" gorm:"column:full_name"`
	Phone     string    `json:"phone,omitempty" gorm:"column:phone"`
	Email     string    `json:"email,omitempty" gorm:"column:email"`
	Address   string    `json:"address,omitempty" gorm:"column:address"`
	Note      string    `json:"note,omitempty" gorm:"column:note"`
	IsActive  bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

type LevelDiscount struct {
	StoreID         string    `json:"store_id" gorm:"column:store_id;primaryKey"`
	Level           int       `json:"level" gorm:"column:level;primaryKey"`
	DiscountPercent float64   `json:"discount_percent" gorm:"column:discount_percent"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at"`
}

type CreateCustomerRequest struct {
	Level    *int   `json:"level"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Address  string `json:"address"`
	Note     string `json:"note"`
	IsActive *bool  `json:"is_active"`
}

type UpdateCustomerRequest struct {
	Level    *int    `json:"level"`
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Address  *string `json:"address"`
	Note     *string `json:"note"`
	IsActive *bool   `json:"is_active"`
}

type UpsertLevelDiscountRequest struct {
	DiscountPercent float64 `json:"discount_percent"`
}

func (Customer) TableName() string {
	return "customers"
}

func (LevelDiscount) TableName() string {
	return "customer_level_discounts"
}
