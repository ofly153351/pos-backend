package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/httpx"
)

// StoreAuthorizer resolves a caller's active store membership once and gates
// store-scoped routes by the required auth.StoreLevel. It owns the store_members
// query so that role resolution is performed in exactly one place; the role ↔
// permission mapping lives in auth.RoleSatisfies (single source of truth).
type StoreAuthorizer struct {
	db *gorm.DB
}

func NewStoreAuthorizer(db *gorm.DB) *StoreAuthorizer {
	return &StoreAuthorizer{db: db}
}

// Require returns a handler that rejects the request unless the caller is an
// active member of the store named by the ":storeID" path parameter whose role
// satisfies the required level. platform_admin bypasses the membership check and
// is treated as owner.
//
// The middleware is fail-closed: a missing :storeID param, a missing membership,
// a suspended membership, or an insufficient role all return 403 FORBIDDEN.
// On success the resolved StoreAccess is attached to the request context so
// services can read the caller's effective store role without re-querying
// store_members.
func (a *StoreAuthorizer) Require(level auth.StoreLevel) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := ClaimsFromContext(c)
		if claims.UserID == "" {
			return httpx.ErrForbidden(c, "forbidden")
		}

		storeID := c.Params("storeID")
		if storeID == "" {
			return httpx.ErrForbidden(c, "missing storeID")
		}

		// platform_admin is global super-admin: bypass membership, act as owner.
		if claims.Role == auth.RolePlatformAdmin {
			access := auth.StoreAccess{StoreID: storeID, Role: auth.RoleOwner}
			c.SetUserContext(auth.WithStoreAccess(c.UserContext(), access))
			return c.Next()
		}

		var member struct {
			Role   string `gorm:"column:role"`
			Status string `gorm:"column:status"`
		}
		err := a.db.WithContext(c.UserContext()).
			Table("store_members").
			Select("role, status").
			Where("store_id = ? AND user_id = ?", storeID, claims.UserID).
			Take(&member).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.ErrForbidden(c, "forbidden")
			}
			return httpx.ErrInternal(c)
		}

		if member.Status != "active" {
			return httpx.ErrForbidden(c, "forbidden")
		}
		if !auth.RoleSatisfies(member.Role, level) {
			return httpx.ErrForbidden(c, "forbidden")
		}

		access := auth.StoreAccess{StoreID: storeID, Role: member.Role}
		c.SetUserContext(auth.WithStoreAccess(c.UserContext(), access))
		return c.Next()
	}
}
