package auth

import "time"

const (
	RolePlatformAdmin = "platform_admin"
	RoleOwner         = "owner"
	RoleManager       = "manager"
	RoleCashier       = "cashier"
)

type User struct {
	ID           string    `json:"id" gorm:"column:id;primaryKey"`
	Name         string    `json:"name" gorm:"column:full_name"`
	Email        string    `json:"email" gorm:"column:email"`
	Role         string    `json:"role" gorm:"column:role"`
	Status       string    `json:"status" gorm:"column:status"`
	PasswordHash string    `json:"-" gorm:"column:password_hash"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
}

func (User) TableName() string {
	return "users"
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User        User   `json:"user"`
	StoreID     string `json:"store_id,omitempty"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}
