// Package lifecycle holds the single canonical decision engine for warehouse and
// storage-location deletion/archive eligibility. Both modules gather their own
// entity-specific dependency counts (different tables) and feed them into the ONE
// pure Assess function here, so the *decision rules* (what blocks, archive vs
// hard-delete, suggested action, error code) live in exactly one place and never
// drift between the two entities or leak into HTTP handlers.
package lifecycle

// EntityType selects the entity-specific error-code vocabulary.
type EntityType string

const (
	EntityWarehouse EntityType = "warehouse"
	EntityLocation  EntityType = "location"
)

// SuggestedAction is the safe strategy a plain "Delete" click should resolve to.
const (
	ActionHardDelete = "hard_delete" // never used, reference-free → permanent row delete
	ActionArchive    = "archive"     // zero-stock + history, no blockers → soft-delete
	ActionBlocked    = "blocked"     // a hard blocker is present → remediation required
)

// Machine-readable blocker codes (mirrored in the frontend TH/EN dictionaries).
const (
	CodeWarehouseSystemProtected     = "WAREHOUSE_SYSTEM_PROTECTED"
	CodeWarehouseHasStock            = "WAREHOUSE_HAS_STOCK"
	CodeWarehouseHasBlockedLocations = "WAREHOUSE_HAS_BLOCKED_LOCATIONS"
	CodeWarehouseHasOpenOperations   = "WAREHOUSE_HAS_OPEN_OPERATIONS"

	CodeLocationSystemProtected   = "LOCATION_SYSTEM_PROTECTED"
	CodeLocationHasStock          = "LOCATION_HAS_STOCK"
	CodeLocationIsProductDefault  = "LOCATION_IS_PRODUCT_DEFAULT"
	CodeLocationHasOpenOperations = "LOCATION_HAS_OPEN_OPERATIONS"

	// CodeEntityStateChanged is returned when the dependency state changed between the
	// pre-check assessment and the locked re-assessment inside the delete transaction.
	CodeEntityStateChanged = "ENTITY_STATE_CHANGED"
)

// BlockerCounts is the entity-agnostic dependency snapshot. For a warehouse the
// stock/product-default/movement/open-operation figures are aggregated across all of
// its child locations; for a location they describe the location itself.
type BlockerCounts struct {
	StockQuantity               int  `json:"stock_quantity"`                 // SUM(stocks.quantity); HARD blocker if != 0
	StockRowCount               int  `json:"stock_row_count"`                // COUNT(stocks rows); history marker (zero-qty rows still count)
	ActiveLocationCount         int  `json:"active_location_count"`          // warehouse only
	LocationCount               int  `json:"location_count"`                 // warehouse only
	ProductDefaultLocationCount int  `json:"product_default_location_count"` // products.default_location_id; HARD blocker
	OpenTransferCount           int  `json:"open_transfer_count"`            // structurally 0 (transfers have no open state); kept for API shape
	OpenReceivingCount          int  `json:"open_receiving_count"`           // receipts draft|pending_review; HARD blocker
	OpenStockCountCount         int  `json:"open_stock_count_count"`         // count sessions draft|counting|review; HARD blocker
	MovementCount               int  `json:"movement_count"`                 // stock_movements (source OR destination); history marker
	HistoricalReferenceCount    int  `json:"historical_reference_count"`     // confirmed/cancelled receipts + warehouse_inventory + receipt items; history marker
	BlockedChildCount           int  `json:"blocked_child_count"`            // warehouse only: children that are themselves hard-blocked or protected
	SystemProtected             bool `json:"system_protected"`               // is_default / is_default_sale / protected child; HARD blocker
}

// Assessment is the canonical verdict returned by the assessment endpoint and used by
// the smart-delete transaction.
type Assessment struct {
	CanHardDelete   bool          `json:"can_hard_delete"`
	CanArchive      bool          `json:"can_archive"`
	CanDeactivate   bool          `json:"can_deactivate"`
	SuggestedAction string        `json:"suggested_action"`
	BlockerCode     string        `json:"blocker_code,omitempty"`
	Blockers        BlockerCounts `json:"blockers"`
}

// hasHistory reports whether the record was ever used. Any surviving stock row (even a
// zero-quantity one), movement, or completed/cancelled reference means a hard delete
// would erase audit-relevant linkage, so archive is the safe strategy instead.
func (b BlockerCounts) hasHistory() bool {
	return b.StockRowCount > 0 || b.MovementCount > 0 || b.HistoricalReferenceCount > 0
}

// Assess applies the canonical decision rules to a dependency snapshot. It is pure
// (no I/O) so it is trivially unit-testable and identical for both call sites and the
// locked re-check inside the delete transaction.
//
// Decision order (first match wins for the blocker code):
//  1. SystemProtected            → blocked  (*_SYSTEM_PROTECTED)
//  2. StockQuantity != 0         → blocked  (*_HAS_STOCK)
//  3. warehouse BlockedChildCount → blocked (WAREHOUSE_HAS_BLOCKED_LOCATIONS)
//  4. location  ProductDefault    → blocked (LOCATION_IS_PRODUCT_DEFAULT)
//  5. OpenReceiving/OpenCount     → blocked (*_HAS_OPEN_OPERATIONS)
//  6. else hasHistory             → archive
//  7. else                        → hard_delete
//
// CanDeactivate is independent of blockers: deactivation keeps the row and merely hides
// it from new operations, so it is allowed for anything that is not system-protected.
func Assess(entity EntityType, b BlockerCounts) Assessment {
	a := Assessment{Blockers: b, CanDeactivate: !b.SystemProtected}

	switch {
	case b.SystemProtected:
		a.SuggestedAction = ActionBlocked
		a.BlockerCode = protectedCode(entity)

	case b.StockQuantity != 0:
		a.SuggestedAction = ActionBlocked
		a.BlockerCode = hasStockCode(entity)

	case entity == EntityWarehouse && b.BlockedChildCount > 0:
		a.SuggestedAction = ActionBlocked
		a.BlockerCode = CodeWarehouseHasBlockedLocations

	case entity == EntityLocation && b.ProductDefaultLocationCount > 0:
		a.SuggestedAction = ActionBlocked
		a.BlockerCode = CodeLocationIsProductDefault

	case b.OpenReceivingCount > 0 || b.OpenStockCountCount > 0:
		a.SuggestedAction = ActionBlocked
		a.BlockerCode = openOpsCode(entity)

	case b.hasHistory():
		a.SuggestedAction = ActionArchive
		a.CanArchive = true

	default:
		a.SuggestedAction = ActionHardDelete
		a.CanHardDelete = true
		a.CanArchive = true // a never-used record can equally be archived if the caller prefers
	}

	return a
}

func protectedCode(e EntityType) string {
	if e == EntityWarehouse {
		return CodeWarehouseSystemProtected
	}
	return CodeLocationSystemProtected
}

func hasStockCode(e EntityType) string {
	if e == EntityWarehouse {
		return CodeWarehouseHasStock
	}
	return CodeLocationHasStock
}

func openOpsCode(e EntityType) string {
	if e == EntityWarehouse {
		return CodeWarehouseHasOpenOperations
	}
	return CodeLocationHasOpenOperations
}
