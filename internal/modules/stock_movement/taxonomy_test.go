package stock_movement

import "testing"

// Phase W5 regression — the movement taxonomy is the single source of truth for
// stock_movements.type. Every type the code can write must be described exactly once,
// with non-empty Thai+English labels and a valid direction, and the paired-transfer
// counting rule must mark exactly the IN leg.

func TestTaxonomy_CoversEveryWrittenType(t *testing.T) {
	written := []string{
		MovementTypeIn, MovementTypeOut, MovementTypeTransfer, MovementTypeTransferOut,
		MovementTypeTransferIn, MovementTypeSale, MovementTypeAdjust, MovementTypeReturn,
		MovementTypeCountCorrection, MovementTypeAllocate,
	}
	for _, code := range written {
		if _, ok := MovementMeta(code); !ok {
			t.Errorf("movement type %q has no taxonomy entry", code)
		}
	}
}

func TestTaxonomy_UniqueAndWellFormed(t *testing.T) {
	seen := make(map[string]bool)
	for _, m := range MovementTaxonomy {
		if seen[m.Code] {
			t.Errorf("duplicate taxonomy code %q", m.Code)
		}
		seen[m.Code] = true
		if m.LabelTH == "" || m.LabelEN == "" {
			t.Errorf("type %q missing a label (th=%q en=%q)", m.Code, m.LabelTH, m.LabelEN)
		}
		switch m.Direction {
		case DirectionIn, DirectionOut, DirectionNeutral:
		default:
			t.Errorf("type %q has invalid direction %q", m.Code, m.Direction)
		}
	}
}

func TestTaxonomy_TransferCountedOnce(t *testing.T) {
	// A logical transfer writes TRANSFER_OUT + TRANSFER_IN sharing one reference_id;
	// only the IN leg is the canonical countable event (no double-count).
	if !IsTransferInLeg(MovementTypeTransferIn) {
		t.Error("TRANSFER_IN must be the countable leg")
	}
	for _, code := range []string{MovementTypeTransferOut, MovementTypeSale, MovementTypeIn} {
		if IsTransferInLeg(code) {
			t.Errorf("%q must not be counted as a transfer leg", code)
		}
	}
}
