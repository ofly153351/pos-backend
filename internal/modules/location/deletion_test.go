package location

import (
	"errors"
	"testing"

	"pos-backend/internal/platform/lifecycle"
)

// TestLocationDeletionDecisions locks in the storage-location safe-delete contract (§4/§19):
// for each dependency snapshot the canonical engine must resolve the right action, and a
// blocked action must map to the right localized sentinel error (the message the handler
// returns while the machine code travels in the structured details).
//
// Pure decision/mapping coverage — no database. The transactional behaviours (idempotent
// double-click, ENTITY_STATE_CHANGED under the lock, FK residual-race on hard delete) are
// exercised live against the disposable database.
func TestLocationDeletionDecisions(t *testing.T) {
	tests := []struct {
		name       string
		b          lifecycle.BlockerCounts
		wantAction string
		wantErr    error // nil for non-blocked actions
	}{
		{
			name:       "1. never used → hard delete",
			b:          lifecycle.BlockerCounts{},
			wantAction: lifecycle.ActionHardDelete,
		},
		{
			name:       "2. zero-stock with history → archive",
			b:          lifecycle.BlockerCounts{StockRowCount: 1, MovementCount: 3, HistoricalReferenceCount: 2},
			wantAction: lifecycle.ActionArchive,
		},
		{
			name:       "3. stock remaining → blocked HAS_STOCK",
			b:          lifecycle.BlockerCounts{StockQuantity: 7, StockRowCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrLocationHasStock,
		},
		{
			name:       "4. product default → blocked IS_PRODUCT_DEFAULT",
			b:          lifecycle.BlockerCounts{ProductDefaultLocationCount: 2, StockRowCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrLocationIsProductDefault,
		},
		{
			name:       "5. open goods receipt → blocked HAS_OPEN_OPERATIONS",
			b:          lifecycle.BlockerCounts{OpenReceivingCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrLocationHasOpenOperations,
		},
		{
			name:       "6. open stock-count session → blocked HAS_OPEN_OPERATIONS",
			b:          lifecycle.BlockerCounts{OpenStockCountCount: 1},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrLocationHasOpenOperations,
		},
		{
			name:       "7. default sale point → blocked SYSTEM_PROTECTED",
			b:          lifecycle.BlockerCounts{SystemProtected: true, StockRowCount: 4, MovementCount: 2},
			wantAction: lifecycle.ActionBlocked,
			wantErr:    ErrDefaultSaleLocationDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := lifecycle.Assess(lifecycle.EntityLocation, tt.b)
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
