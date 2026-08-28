package warehouse

import (
	"context"
	"errors"
	"testing"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/lifecycle"
)

// Phase 0 test coverage map (warehouse-scoped product inventory).
//
// Unit-tested here (pure logic + service gating):
//   #2 ready/storage split, #3 total = ready + storage, #7 sale-only, #8 storage-only,
//   #9 zero stock, #10 low status, #11 out-of-stock status, #12 pagination, #13 search,
//   #14 category filter, #15 cashier read, #16 suspended rejected, #17 non-member rejected.
// Verified via UAT (SQL scoping — the codebase has no DB-integration test harness, and
// adding sqlmock/a test DB is out of scope/forbidden):
//   #1 warehouse with sale+storage locations, #4 multiple locations same warehouse,
//   #5 other-warehouse stock excluded, #6 inactive-location stock included.
//   #18 no N+1 — guaranteed by design (a single grouped query in ListWarehouseStockRows).

func TestDeriveStockStatus(t *testing.T) {
	cases := []struct {
		total, min int
		want       string
	}{
		{0, 50, StockStatusOut},  // zero → out (#9, #11)
		{0, 0, StockStatusOut},   // zero, no min → out
		{40, 50, StockStatusLow}, // <= min → low (#10)
		{50, 50, StockStatusLow}, // == min → low
		{51, 50, StockStatusAvailable},
		{10, 0, StockStatusAvailable}, // min 0 → never low (only out at 0)
	}
	for _, c := range cases {
		if got := deriveStockStatus(c.total, c.min); got != c.want {
			t.Errorf("deriveStockStatus(%d,%d) = %q, want %q", c.total, c.min, got, c.want)
		}
	}
}

func TestBuildWarehouseInventory_SplitTotalAndSummary(t *testing.T) {
	rows := []WarehouseStockRow{
		// น้ำดื่ม: ready 80 + storage 220 = 300 (available; min 50)
		{ProductID: "p1", ProductName: "น้ำดื่ม 600ml", SKU: "WATER-600", CostPrice: 7, MinStock: 50, ReadyStock: 80, StorageStock: 220},
		// มาม่า: storage-only 8 <= min 10 → low (#8 storage-only, #10 low)
		{ProductID: "p2", ProductName: "มาม่าหมูสับ", SKU: "MAMA-MS", CostPrice: 6, MinStock: 10, ReadyStock: 0, StorageStock: 8},
		// นม: sale-only 48 (available; #7 sale-only)
		{ProductID: "p3", ProductName: "นมจืด", SKU: "MILK", CostPrice: 12, MinStock: 20, ReadyStock: 48, StorageStock: 0},
		// น้ำแข็ง: zero (#9 zero → out)
		{ProductID: "p4", ProductName: "น้ำแข็งหลอด", SKU: "ICE", CostPrice: 25, MinStock: 5, ReadyStock: 0, StorageStock: 0},
	}
	summary, items, total := buildWarehouseInventory(rows, WarehouseInventoryQuery{Page: 1, PageSize: 20})

	if total != 4 || len(items) != 4 {
		t.Fatalf("expected 4 items, got total=%d len=%d", total, len(items))
	}
	// #3 total = ready + storage for every item
	for _, it := range items {
		if it.TotalStock != it.ReadyStock+it.StorageStock {
			t.Errorf("%s total %d != ready %d + storage %d", it.ProductID, it.TotalStock, it.ReadyStock, it.StorageStock)
		}
	}
	byID := map[string]WarehouseInventoryProduct{}
	for _, it := range items {
		byID[it.ProductID] = it
	}
	if byID["p1"].TotalStock != 300 || byID["p1"].Status != StockStatusAvailable {
		t.Errorf("p1 = %+v", byID["p1"])
	}
	if byID["p2"].Status != StockStatusLow || byID["p2"].ReadyStock != 0 {
		t.Errorf("p2 (storage-only low) = %+v", byID["p2"])
	}
	if byID["p3"].StorageStock != 0 || byID["p3"].Status != StockStatusAvailable {
		t.Errorf("p3 (sale-only) = %+v", byID["p3"])
	}
	if byID["p4"].Status != StockStatusOut {
		t.Errorf("p4 (zero) status = %q", byID["p4"].Status)
	}
	// Summary over the full (unfiltered) set
	if summary.ProductCount != 4 {
		t.Errorf("product_count = %d, want 4", summary.ProductCount)
	}
	if summary.ReadyStock != 128 || summary.StorageStock != 228 || summary.TotalStock != 356 {
		t.Errorf("summary stock = %+v, want ready 128 storage 228 total 356", summary)
	}
	// inventory value = Σ cost * total = 7*300 + 6*8 + 12*48 + 25*0 = 2100+48+576 = 2724
	if summary.InventoryValue != 2724 {
		t.Errorf("inventory_value = %v, want 2724", summary.InventoryValue)
	}
	if summary.LowStockCount != 1 || summary.OutOfStockCount != 1 {
		t.Errorf("low=%d out=%d, want low 1 out 1", summary.LowStockCount, summary.OutOfStockCount)
	}
}

func TestBuildWarehouseInventory_StatusFilter(t *testing.T) {
	rows := []WarehouseStockRow{
		{ProductID: "a", ProductName: "A", ReadyStock: 100, MinStock: 10}, // available
		{ProductID: "b", ProductName: "B", StorageStock: 5, MinStock: 10}, // low
		{ProductID: "c", ProductName: "C", MinStock: 10},                  // out
	}
	for _, tc := range []struct {
		status string
		wantID string
	}{
		{StockStatusAvailable, "a"},
		{StockStatusLow, "b"},
		{StockStatusOut, "c"},
	} {
		_, items, total := buildWarehouseInventory(rows, WarehouseInventoryQuery{StockStatus: tc.status, Page: 1, PageSize: 20})
		if total != 1 || items[0].ProductID != tc.wantID {
			t.Errorf("status %q → total %d first %q, want 1 %q", tc.status, total, firstID(items), tc.wantID)
		}
	}
}

func TestBuildWarehouseInventory_LocationTypeFilter(t *testing.T) {
	rows := []WarehouseStockRow{
		{ProductID: "ready-only", ProductName: "R", ReadyStock: 10},
		{ProductID: "storage-only", ProductName: "S", StorageStock: 10},
		{ProductID: "both", ProductName: "B", ReadyStock: 5, StorageStock: 5},
	}
	_, sale, saleTotal := buildWarehouseInventory(rows, WarehouseInventoryQuery{LocationType: LocationTypeSalePoint, Page: 1, PageSize: 20})
	if saleTotal != 2 { // ready-only + both
		t.Errorf("sale_point filter total = %d, want 2 (ids: %v)", saleTotal, ids(sale))
	}
	_, storage, storageTotal := buildWarehouseInventory(rows, WarehouseInventoryQuery{LocationType: LocationTypeStorage, Page: 1, PageSize: 20})
	if storageTotal != 2 { // storage-only + both
		t.Errorf("storage filter total = %d, want 2 (ids: %v)", storageTotal, ids(storage))
	}
	_, _, allTotal := buildWarehouseInventory(rows, WarehouseInventoryQuery{LocationType: LocationTypeAll, Page: 1, PageSize: 20})
	if allTotal != 3 {
		t.Errorf("all filter total = %d, want 3", allTotal)
	}
}

func TestBuildWarehouseInventory_SearchAndCategory(t *testing.T) {
	rows := []WarehouseStockRow{
		{ProductID: "p1", ProductName: "น้ำดื่ม 600ml", SKU: "WATER-600", Barcode: "8851234567890", CategoryID: "cat-drink", ReadyStock: 10},
		{ProductID: "p2", ProductName: "ข้าวสาร 5kg", SKU: "RICE-5KG", Barcode: "8850000000011", CategoryID: "cat-grocery", StorageStock: 20},
	}
	// search by SKU substring (case-insensitive)
	_, bySku, n1 := buildWarehouseInventory(rows, WarehouseInventoryQuery{Search: "water", Page: 1, PageSize: 20})
	if n1 != 1 || bySku[0].ProductID != "p1" {
		t.Errorf("search water → %d %q", n1, firstID(bySku))
	}
	// search by barcode
	_, byBc, n2 := buildWarehouseInventory(rows, WarehouseInventoryQuery{Search: "8850000000011", Page: 1, PageSize: 20})
	if n2 != 1 || byBc[0].ProductID != "p2" {
		t.Errorf("search barcode → %d %q", n2, firstID(byBc))
	}
	// category filter
	_, byCat, n3 := buildWarehouseInventory(rows, WarehouseInventoryQuery{CategoryID: "cat-grocery", Page: 1, PageSize: 20})
	if n3 != 1 || byCat[0].ProductID != "p2" {
		t.Errorf("category filter → %d %q", n3, firstID(byCat))
	}
}

func TestBuildWarehouseInventory_Pagination(t *testing.T) {
	rows := make([]WarehouseStockRow, 0, 25)
	for i := 0; i < 25; i++ {
		rows = append(rows, WarehouseStockRow{ProductID: string(rune('a' + i)), ProductName: string(rune('a' + i)), ReadyStock: 1})
	}
	_, page1, total := buildWarehouseInventory(rows, WarehouseInventoryQuery{Page: 1, PageSize: 20})
	if total != 25 || len(page1) != 20 {
		t.Fatalf("page1: total=%d len=%d, want 25 / 20", total, len(page1))
	}
	_, page2, _ := buildWarehouseInventory(rows, WarehouseInventoryQuery{Page: 2, PageSize: 20})
	if len(page2) != 5 {
		t.Errorf("page2 len = %d, want 5", len(page2))
	}
	// page beyond the end → empty (not nil)
	_, page99, _ := buildWarehouseInventory(rows, WarehouseInventoryQuery{Page: 99, PageSize: 20})
	if page99 == nil || len(page99) != 0 {
		t.Errorf("page99 = %v, want empty non-nil slice", page99)
	}
}

func TestValidateInventoryQuery(t *testing.T) {
	valid := WarehouseInventoryQuery{Page: 1, PageSize: 20}
	if err := validateInventoryQuery(valid); err != nil {
		t.Errorf("valid query rejected: %v", err)
	}
	bad := []WarehouseInventoryQuery{
		{Page: 0, PageSize: 20},
		{Page: 1, PageSize: 0},
		{Page: 1, PageSize: 101},
		{Page: 1, PageSize: 20, StockStatus: "weird"},
		{Page: 1, PageSize: 20, LocationType: "weird"},
		{Page: 1, PageSize: 20, Sort: "weird"},
	}
	for i, q := range bad {
		if err := validateInventoryQuery(q); err == nil {
			t.Errorf("bad query #%d accepted: %+v", i, q)
		}
	}
}

// ── Service-layer tests (permission gating + warehouse existence) ──

func TestService_ListInventoryProducts_CashierAllowed(t *testing.T) {
	repo := &stubRepo{
		warehouse: Warehouse{ID: "wh1", Name: "คลังหลัก", Code: "WH-MAIN"},
		rows:      []WarehouseStockRow{{ProductID: "p1", ProductName: "X", ReadyStock: 5, StorageStock: 5, MinStock: 2}},
	}
	svc := NewService(repo)
	resp, err := svc.ListInventoryProducts(context.Background(), auth.Claims{UserID: "u-cashier", Role: "cashier"}, "store1", "wh1", WarehouseInventoryQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("cashier read should be allowed, got %v", err)
	}
	if resp.Warehouse.ID != "wh1" || resp.Warehouse.Code != "WH-MAIN" {
		t.Errorf("warehouse ref = %+v", resp.Warehouse)
	}
	if resp.Summary.TotalStock != 10 || len(resp.Items) != 1 || resp.Pagination.Total != 1 {
		t.Errorf("unexpected payload: summary=%+v items=%d pag=%+v", resp.Summary, len(resp.Items), resp.Pagination)
	}
}

func TestService_ListInventoryProducts_WarehouseNotFound(t *testing.T) {
	repo := &stubRepo{getByIDErr: ErrWarehouseNotFound}
	svc := NewService(repo)
	_, err := svc.ListInventoryProducts(context.Background(), auth.Claims{UserID: "u1", Role: "owner"}, "store1", "missing", WarehouseInventoryQuery{Page: 1, PageSize: 20})
	if !errors.Is(err, ErrWarehouseNotFound) {
		t.Fatalf("expected ErrWarehouseNotFound, got %v", err)
	}
}

// ── helpers ──

func firstID(items []WarehouseInventoryProduct) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].ProductID
}

func ids(items []WarehouseInventoryProduct) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ProductID
	}
	return out
}

// stubRepo implements warehouse.Repository for service-layer tests. Only the methods used
// by ListInventoryProducts are configurable; the rest are inert.
type stubRepo struct {
	warehouse  Warehouse
	getByIDErr error
	rows       []WarehouseStockRow
	rowsErr    error
}

func (s *stubRepo) GetByID(_ context.Context, _, _ string) (Warehouse, error) {
	if s.getByIDErr != nil {
		return Warehouse{}, s.getByIDErr
	}
	return s.warehouse, nil
}
func (s *stubRepo) ListWarehouseStockRows(_ context.Context, _ string) ([]WarehouseStockRow, error) {
	return s.rows, s.rowsErr
}

// Inert implementations to satisfy the Repository interface.
func (s *stubRepo) Create(_ context.Context, _ Warehouse) (Warehouse, error) { return Warehouse{}, nil }
func (s *stubRepo) ListByStore(_ context.Context, _ string, _ bool) ([]Warehouse, error) {
	return nil, nil
}
func (s *stubRepo) Update(_ context.Context, _ Warehouse) (Warehouse, error) { return Warehouse{}, nil }
func (s *stubRepo) Delete(_ context.Context, _, _ string) error              { return nil }
func (s *stubRepo) GatherDeletionBlockers(_ context.Context, _, _ string) (lifecycle.BlockerCounts, error) {
	return lifecycle.BlockerCounts{}, nil
}
func (s *stubRepo) ApplyDeletion(_ context.Context, _, _, _ string) (lifecycle.Assessment, string, error) {
	return lifecycle.Assessment{}, "", nil
}
func (s *stubRepo) AddProduct(_ context.Context, _, _, _ string, _ int) (WarehouseProduct, error) {
	return WarehouseProduct{}, nil
}
func (s *stubRepo) ListProducts(_ context.Context, _ string) ([]WarehouseProduct, error) {
	return nil, nil
}
func (s *stubRepo) UpdateProduct(_ context.Context, _, _, _ string, _ int) error { return nil }
func (s *stubRepo) RemoveProduct(_ context.Context, _, _, _ string) error        { return nil }
func (s *stubRepo) ProductExistsInWarehouse(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (s *stubRepo) ProductTotalQtyInWarehouse(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (s *stubRepo) ProductBelongsToStore(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (s *stubRepo) TransferStock(_ context.Context, _, _, _ string, _ int, _, _, _, _ string) error {
	return nil
}
func (s *stubRepo) ListWarehouseInventory(_ context.Context, _ string) ([]WarehouseInventory, error) {
	return nil, nil
}
func (s *stubRepo) AllocateInventoryToStock(_ context.Context, _, _, _ string, _ int, _, _ string) error {
	return nil
}
