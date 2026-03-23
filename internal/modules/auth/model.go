package auth

import "time"

const (
	RolePlatformAdmin = "platform_admin"
	RoleOwner         = "owner"
	RoleManager       = "manager"
	RoleCashier       = "cashier"
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
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
