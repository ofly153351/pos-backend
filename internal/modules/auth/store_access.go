package auth

import "context"

// StoreAccess is the caller's resolved membership for the store named in the
// request path. The store authorization middleware (StoreAuthorizer) resolves it
// once against store_members and stores it in the request context; services read
// it back with StoreAccessFromContext instead of re-querying store_members.
type StoreAccess struct {
	StoreID string
	// Role is the active store_members.role for this store ("owner", "manager",
	// "cashier", "warehouse"). A platform_admin is normalised to "owner" so
	// role-based business logic treats it as full power. "" means no active
	// membership (or the value was never resolved).
	Role string
}

// CanManage reports whether the caller may perform management actions
// (owner or manager) for this store.
func (a StoreAccess) CanManage() bool {
	return a.Role == RoleOwner || a.Role == RoleManager
}

// IsOwner reports whether the caller is the store owner (or a platform_admin,
// which is normalised to owner).
func (a StoreAccess) IsOwner() bool {
	return a.Role == RoleOwner
}

type storeAccessCtxKey struct{}

// WithStoreAccess attaches a resolved StoreAccess to ctx.
func WithStoreAccess(ctx context.Context, access StoreAccess) context.Context {
	return context.WithValue(ctx, storeAccessCtxKey{}, access)
}

// StoreAccessFromContext returns the resolved StoreAccess from ctx, or the zero
// value when no store authorization middleware ran (e.g. user-scoped routes).
func StoreAccessFromContext(ctx context.Context) StoreAccess {
	access, _ := ctx.Value(storeAccessCtxKey{}).(StoreAccess)
	return access
}
