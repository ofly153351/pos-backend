package auth

// StoreLevel is the canonical permission level required to reach a store-scoped
// route. The four levels form a strict chain, each a superset of the previous:
//
//	owner  → {owner}
//	manage → {owner, manager}
//	operate→ {owner, manager, cashier}
//	access → {owner, manager, cashier, warehouse}
//
// This is the single source of truth for the store role ↔ permission mapping,
// replacing the per-module UserCanManageStore / UserCanOperateStore /
// UserHasStoreAccess / canManage / canView / ensure* helpers that previously
// duplicated the same store_members query with slightly different role lists.
type StoreLevel int

const (
	StoreLevelOwner   StoreLevel = iota // owner only
	StoreLevelManage                    // owner, manager
	StoreLevelOperate                   // owner, manager, cashier
	StoreLevelAccess                    // owner, manager, cashier, warehouse
)

// RoleSatisfies reports whether an active store role satisfies the given level.
// Callers that cannot rely on the store-authorization middleware (e.g. resources
// whose storeID is not in the URL path) use this to keep the role→level decision
// in one place.
func RoleSatisfies(role string, level StoreLevel) bool {
	switch level {
	case StoreLevelOwner:
		return role == RoleOwner
	case StoreLevelManage:
		return role == RoleOwner || role == RoleManager
	case StoreLevelOperate:
		return role == RoleOwner || role == RoleManager || role == RoleCashier
	case StoreLevelAccess:
		return role == RoleOwner || role == RoleManager || role == RoleCashier || role == RoleWarehouse
	default:
		return false
	}
}
