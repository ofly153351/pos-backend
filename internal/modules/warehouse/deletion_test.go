package warehouse

import (
	"errors"
	"testing"

	"pos-backend/internal/platform/lifecycle"
)

// TestWarehouseDeletionDecisions locks in the warehouse safe-delete contract (§5/§19): for
// each dependency snapshot, the canonical engine must resolve the right action, and a
// blocked action must map to the right localized sentinel error (the message the handler
// returns while the machine code travels in the structured details).
//
// Pure decision/mapping coverage — no database. The transactional behaviours (atomic
// archive of warehouse + children, idempotent double-click, ENTITY_STATE_CHANGED under the
// lock, FK residual-race) are exercised live against the disposable database.
func TestWarehouseDeletionDecisions(t *testing.T) {
	tests := []struct {
		name       string
		b          lifecycle.BlockerCounts
		wantAction string
		wantErr    error // nil for non-blocked actions
	}{
		{
			name:       "1. never used → hard delete",
			b:          lifecycle.BlockerCounts{LocationCount: 0},
			wantAction: lifecycle.ActionHardDelete,
		},
		{
			name:       "2. zero-stock children with history → archive",
			b:          lifecycle.BlockerCounts{LocationCount: 1, StockRowCount: 1, MovementCount: 2, HistoricalReferenceCount: 1},
			wantAction: lifecycle.ActionArchive,
		},
		{
			name:       "3. child stock remaining → blocked HAS_STOCK",
			b:          lifecycle.BlockerCounts{LocationCount: 1, StockQuantity: 12, StockRowCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrWarehouseHasStock,
		},
		{
			name:       "4. blocked child (product default) → blocked HAS_BLOCKED_LOCATIONS",
			b:          lifecycle.BlockerCounts{LocationCount: 2, BlockedChildCount: 1, ProductDefaultLocationCount: 1, StockRowCount: 2},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrWarehouseHasBlockedLocations,
		},
		{
			name:       "5. open goods receipt → blocked HAS_OPEN_OPERATIONS",
			b:          lifecycle.BlockerCounts{LocationCount: 1, OpenReceivingCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrWarehouseHasOpenOperations,
		},
		{
			name:       "6. open stock-count session → blocked HAS_OPEN_OPERATIONS",
			b:          lifecycle.BlockerCounts{LocationCount: 1, OpenStockCountCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrWarehouseHasOpenOperations,
		},
		{
			name:       "7. default warehouse → blocked SYSTEM_PROTECTED",
			b:          lifecycle.BlockerCounts{SystemProtected: true, LocationCount: 2, StockRowCount: 4},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrDefaultWarehouseDelete,
		},
		{
			name:       "8. child is store default sale point → blocked SYSTEM_PROTECTED",
			b:          lifecycle.BlockerCounts{SystemProtected: true, LocationCount: 1, BlockedChildCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrDefaultWarehouseDelete,
		},
		{
			name:       "9. stock outranks open operations → HAS_STOCK",
			b:          lifecycle.BlockerCounts{LocationCount: 1, StockQuantity: 1, StockRowCount: 1, OpenReceivingCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrWarehouseHasStock,
		},
		{
			name:       "10. protection outranks stock → SYSTEM_PROTECTED",
			b:          lifecycle.BlockerCounts{SystemProtected: true, StockQuantity: 99, StockRowCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrDefaultWarehouseDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := lifecycle.Assess(lifecycle.EntityWarehouse, tt.b)
			if a.SuggestedAction != tt.wantAction {
				t.Fatalf("SuggestedAction = %q, want %q", a.SuggestedAction, tt.wantAction)
			}
			if tt.wantAction != lifecycle.ActionBlocked {
				if a.BlockerCode != "" {
					t.Errorf("non-blocked action carried BlockerCode %q", a.BlockerCode)
				}
				return
			}
			gotErr := blockerError(a.BlockerCode)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("blockerError(%q) = %v, want %v", a.BlockerCode, gotErr, tt.wantErr)
			}
		})
	}
}
