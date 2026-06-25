package lifecycle

import "testing"

func TestAssess(t *testing.T) {
	tests := []struct {
		name       string
		entity     EntityType
		b          BlockerCounts
		wantAction string
		wantCode   string
		wantHard   bool
		wantArch   bool
		wantDeact  bool
	}{
		// ── Location ─────────────────────────────────────────────────────────────
		{
			name:       "location never used → hard delete",
			entity:     EntityLocation,
			b:          BlockerCounts{},
			wantAction: ActionHardDelete, wantCode: "", wantHard: true, wantArch: true, wantDeact: true,
		},
		{
			name:       "location zero-stock with history → archive",
			entity:     EntityLocation,
			b:          BlockerCounts{StockRowCount: 1, MovementCount: 1},
			wantAction: ActionArchive, wantCode: "", wantHard: false, wantArch: true, wantDeact: true,
		},
		{
			name:       "location with stock → blocked HAS_STOCK",
			entity:     EntityLocation,
			b:          BlockerCounts{StockQuantity: 5, StockRowCount: 1},
			wantAction: ActionBlocked, wantCode: CodeLocationHasStock, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "location is product default → blocked IS_PRODUCT_DEFAULT",
			entity:     EntityLocation,
			b:          BlockerCounts{ProductDefaultLocationCount: 3, StockRowCount: 2},
			wantAction: ActionBlocked, wantCode: CodeLocationIsProductDefault, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "location open receiving → blocked HAS_OPEN_OPERATIONS",
			entity:     EntityLocation,
			b:          BlockerCounts{OpenReceivingCount: 1},
			wantAction: ActionBlocked, wantCode: CodeLocationHasOpenOperations, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "location open count session → blocked HAS_OPEN_OPERATIONS",
			entity:     EntityLocation,
			b:          BlockerCounts{OpenStockCountCount: 1},
			wantAction: ActionBlocked, wantCode: CodeLocationHasOpenOperations, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "location default-sale protected → blocked SYSTEM_PROTECTED, no deactivate",
			entity:     EntityLocation,
			b:          BlockerCounts{SystemProtected: true, StockRowCount: 5},
			wantAction: ActionBlocked, wantCode: CodeLocationSystemProtected, wantHard: false, wantArch: false, wantDeact: false,
		},
		// ── Warehouse ────────────────────────────────────────────────────────────
		{
			name:       "warehouse empty/never used → hard delete",
			entity:     EntityWarehouse,
			b:          BlockerCounts{LocationCount: 0},
			wantAction: ActionHardDelete, wantCode: "", wantHard: true, wantArch: true, wantDeact: true,
		},
		{
			name:       "warehouse zero-stock children with history → archive",
			entity:     EntityWarehouse,
			b:          BlockerCounts{LocationCount: 1, StockRowCount: 1, MovementCount: 1},
			wantAction: ActionArchive, wantCode: "", wantHard: false, wantArch: true, wantDeact: true,
		},
		{
			name:       "warehouse child stock → blocked HAS_STOCK",
			entity:     EntityWarehouse,
			b:          BlockerCounts{StockQuantity: 73, StockRowCount: 1, LocationCount: 1},
			wantAction: ActionBlocked, wantCode: CodeWarehouseHasStock, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "warehouse blocked child (product default) → blocked HAS_BLOCKED_LOCATIONS",
			entity:     EntityWarehouse,
			b:          BlockerCounts{BlockedChildCount: 1, ProductDefaultLocationCount: 2, LocationCount: 2, StockRowCount: 3},
			wantAction: ActionBlocked, wantCode: CodeWarehouseHasBlockedLocations, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "warehouse open child receiving → blocked HAS_OPEN_OPERATIONS",
			entity:     EntityWarehouse,
			b:          BlockerCounts{OpenReceivingCount: 1, LocationCount: 1},
			wantAction: ActionBlocked, wantCode: CodeWarehouseHasOpenOperations, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "warehouse default protected → blocked SYSTEM_PROTECTED, no deactivate",
			entity:     EntityWarehouse,
			b:          BlockerCounts{SystemProtected: true, LocationCount: 2},
			wantAction: ActionBlocked, wantCode: CodeWarehouseSystemProtected, wantHard: false, wantArch: false, wantDeact: false,
		},
		{
			name:       "stock blocker outranks open-operation code",
			entity:     EntityWarehouse,
			b:          BlockerCounts{StockQuantity: 1, OpenReceivingCount: 1, StockRowCount: 1},
			wantAction: ActionBlocked, wantCode: CodeWarehouseHasStock, wantHard: false, wantArch: false, wantDeact: true,
		},
		{
			name:       "protected outranks stock code",
			entity:     EntityWarehouse,
			b:          BlockerCounts{SystemProtected: true, StockQuantity: 99, StockRowCount: 1},
			wantAction: ActionBlocked, wantCode: CodeWarehouseSystemProtected, wantHard: false, wantArch: false, wantDeact: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Assess(tt.entity, tt.b)
			if got.SuggestedAction != tt.wantAction {
				t.Errorf("SuggestedAction = %q, want %q", got.SuggestedAction, tt.wantAction)
			}
			if got.BlockerCode != tt.wantCode {
				t.Errorf("BlockerCode = %q, want %q", got.BlockerCode, tt.wantCode)
			}
			if got.CanHardDelete != tt.wantHard {
				t.Errorf("CanHardDelete = %v, want %v", got.CanHardDelete, tt.wantHard)
			}
			if got.CanArchive != tt.wantArch {
				t.Errorf("CanArchive = %v, want %v", got.CanArchive, tt.wantArch)
			}
			if got.CanDeactivate != tt.wantDeact {
				t.Errorf("CanDeactivate = %v, want %v", got.CanDeactivate, tt.wantDeact)
			}
		})
	}
}
