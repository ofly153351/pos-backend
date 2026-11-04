package p0postgres

import (
	"context"
	"testing"
	"time"

	"pos-backend/internal/modules/dashboard"
	"pos-backend/internal/modules/finance"
	"pos-backend/internal/modules/warehouse_dashboard"
)

// B14: equal scopes give equal, traceable totals across P&L, dashboard and
// warehouse; revenue is pre-VAT from sales; COGS uses sale-time cost only.
func TestPostgresB14ReportsReconcile(t *testing.T) {
	db := openFixture(t)
	app := appFor(db)
	s1 := checkout(t, app, "a", nil, 0, 200, "b14-1", 201) // 2 × 100, cost 40
	s2 := checkout(t, app, "a", nil, 0, 200, "b14-2", 201)
	// Simulate a VAT-inclusive sale (200 + 14 VAT) and a legacy line without a
	// sale-time cost on the second sale.
	exec(t, db, `UPDATE sales SET total_amount = 214, vat_amount = 14 WHERE id = ?`, s1.ID)
	exec(t, db, `UPDATE sale_items SET unit_cost = 0 WHERE sale_id = ?`, s2.ID)

	ctx := context.Background()
	from, to := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	fin := finance.NewPostgresRepository(db)
	rev, err := fin.GetRevenue(ctx, "a", from, to)
	if err != nil {
		t.Fatal(err)
	}
	dash, err := dashboard.NewPostgresRepository(db).GetSummary(ctx, "a", from, to)
	if err != nil {
		t.Fatal(err)
	}
	if rev.NetSales-rev.RefundsPreVAT != 400 || dash.NetSales != 400 {
		t.Fatalf("pre-VAT revenue: finance=%v dashboard=%v, want 400 both", rev.NetSales-rev.RefundsPreVAT, dash.NetSales)
	}

	cogs, err := fin.GetCOGS(ctx, "a", from, to)
	if err != nil {
		t.Fatal(err)
	}
	if cogs.Total != 80 || cogs.MissingCostLines != 1 {
		t.Fatalf("COGS = %+v, want 80 with 1 missing-cost line (no fallback to current cost)", cogs)
	}

	snap, err := fin.GetInventorySnapshot(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	kpi, err := warehouse_dashboard.NewPostgresRepository(db).GetKPI(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if snap.InventoryValue != kpi.StockValue || snap.InventoryValue != 16*40 {
		t.Fatalf("inventory value finance=%v warehouse=%v, want %v", snap.InventoryValue, kpi.StockValue, 16*40)
	}
}
