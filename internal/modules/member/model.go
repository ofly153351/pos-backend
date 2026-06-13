package member

import "time"

// Member is the per-store staff view: a store_members row joined to the user it
// references. Role/Status are the STORE-scoped role and membership status (they
// live on store_members, not on users — a person can belong to several stores
// with a different role/status in each).
type Member struct {
	UserID    string    `json:"user_id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Valid store roles and member statuses (mirror the store_members CHECK
// constraints in migration 031).
const (
	RoleOwner     = "owner"
	RoleManager   = "manager"
	RoleCashier   = "cashier"
	RoleWarehouse = "warehouse"

	StatusActive    = "active"
	StatusSuspended = "suspended"
)

func isValidRole(role string) bool {
	switch role {
	case RoleOwner, RoleManager, RoleCashier, RoleWarehouse:
		return true
	default:
		return false
	}
}

func isValidStatus(status string) bool {
	return status == StatusActive || status == StatusSuspended
}

// AddMemberRequest adds a staff member. If a user with Email already exists, that
// user is reused and only a membership is created (Name/Password are ignored);
// otherwise a new user is created with the given Name/Password and a global
// users.role of 'cashier' (store power comes from Role on the membership).
type AddMemberRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateMemberRequest changes a member's store role and/or membership status.
// Both fields are optional (pointer) so the caller can change either independently.
type UpdateMemberRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}
