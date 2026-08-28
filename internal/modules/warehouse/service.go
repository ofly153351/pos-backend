package warehouse

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/lifecycle"
)

// DeleteOutcome is the result of a smart delete: the action that was applied ("archived" or
// "deleted") plus the dependency assessment that drove it (for the client toast/log).
type DeleteOutcome struct {
	Action     string               `json:"action"`
	Assessment lifecycle.Assessment `json:"assessment"`
}

type Service struct{ repo Repository }

func NewService(repo Repository) Service { return Service{repo: repo} }

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateWarehouseRequest) (Warehouse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return Warehouse{}, ErrInvalidName
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	item := Warehouse{
		ID:          newID(),
		StoreID:     storeID,
		Name:        strings.TrimSpace(req.Name),
		Code:        strings.TrimSpace(req.Code),
		Address:     strings.TrimSpace(req.Address),
		Phone:       strings.TrimSpace(req.Phone),
		ContactName: strings.TrimSpace(req.ContactName),
		IsActive:    isActive,
		CreatedAt:   time.Now().UTC(),
	}
	return s.repo.Create(ctx, item)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string, includeArchived bool) ([]Warehouse, error) {
	return s.repo.ListByStore(ctx, storeID, includeArchived)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, id string) (Warehouse, error) {
	return s.repo.GetByID(ctx, storeID, id)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateWarehouseRequest) (Warehouse, error) {
	item, err := s.repo.GetByID(ctx, storeID, id)
	if err != nil {
		return Warehouse{}, err
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if strings.TrimSpace(item.Name) == "" {
		return Warehouse{}, ErrInvalidName
	}
	if req.Code != nil {
		item.Code = strings.TrimSpace(*req.Code)
	}
	if req.Address != nil {
		item.Address = strings.TrimSpace(*req.Address)
	}
	if req.Phone != nil {
		item.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.ContactName != nil {
		item.ContactName = strings.TrimSpace(*req.ContactName)
	}
	if req.IsActive != nil {
		// Phase W1 invariant: the store's default warehouse must stay active. Disabling
		// it would orphan the default structure, so require choosing a new default first.
		if item.IsDefault && item.IsActive && !*req.IsActive {
			return Warehouse{}, ErrDefaultWarehouseDeactivate
		}
		item.IsActive = *req.IsActive
	}
	item.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, item)
}

// AssessDeletion returns the read-only deletion assessment for a warehouse (drives the
// adaptive delete/archive modal). Manage-level access required.
func (s Service) AssessDeletion(ctx context.Context, actor auth.Claims, storeID, id string) (lifecycle.Assessment, error) {
	// Confirm the warehouse exists and is in this store (→ 404 otherwise).
	if _, err := s.repo.GetByID(ctx, storeID, id); err != nil {
		return lifecycle.Assessment{}, err
	}
	b, err := s.repo.GatherDeletionBlockers(ctx, storeID, id)
	if err != nil {
		return lifecycle.Assessment{}, err
	}
	return lifecycle.Assess(lifecycle.EntityWarehouse, b), nil
}

// Delete is the smart delete (§6): it resolves the safe strategy and applies it atomically.
// A never-used warehouse is hard-deleted; a used warehouse with zero stock and history is
// archived (soft-deleted with its child locations); anything with a hard blocker returns the
// structured blocker error with no mutation. expected is an optional client-declared action
// for optimistic concurrency (ENTITY_STATE_CHANGED on mismatch).
func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, id, expected string) (DeleteOutcome, error) {
	// Confirm existence + store scope before the locked apply (→ 404 otherwise).
	if _, err := s.repo.GetByID(ctx, storeID, id); err != nil {
		return DeleteOutcome{}, err
	}
	assessment, applied, err := s.repo.ApplyDeletion(ctx, storeID, id, expected)
	if err != nil {
		return DeleteOutcome{Assessment: assessment}, err
	}
	return DeleteOutcome{Action: applied, Assessment: assessment}, nil
}

// ──────────────────────────────────────────────
// Warehouse-Product service methods
// ──────────────────────────────────────────────

func (s Service) AddProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID string, req AddWarehouseProductRequest) (WarehouseProduct, error) {
	// Phase W0: the legacy direct add changed stocks without a movement and could
	// silently auto-create locations. It is disabled — receiving now goes through
	// the canonical Goods Receipt workflow.
	return WarehouseProduct{}, ErrWarehouseDirectStockDisabled
}

func (s Service) ListProducts(ctx context.Context, actor auth.Claims, storeID, warehouseID string) ([]WarehouseProduct, error) {

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return nil, err
	}

	return s.repo.ListProducts(ctx, warehouseID)
}

// ListInventoryProducts returns warehouse-scoped product inventory (พร้อมขาย / พื้นที่จัดเก็บ /
// รวมในคลัง) for the selected warehouse. Read-only and operate-level: any active store member
// (owner/manager/cashier/warehouse) may read; suspended/non-members get ErrForbiddenStoreAccess
// (403). The query is assumed already validated/normalized by the handler.
func (s Service) ListInventoryProducts(ctx context.Context, actor auth.Claims, storeID, warehouseID string, q WarehouseInventoryQuery) (WarehouseInventoryResponse, error) {

	// Verify warehouse belongs to store (→ ErrWarehouseNotFound / 404 otherwise).
	wh, err := s.repo.GetByID(ctx, storeID, warehouseID)
	if err != nil {
		return WarehouseInventoryResponse{}, err
	}

	rows, err := s.repo.ListWarehouseStockRows(ctx, warehouseID)
	if err != nil {
		return WarehouseInventoryResponse{}, err
	}

	summary, items, total := buildWarehouseInventory(rows, q)
	return WarehouseInventoryResponse{
		Warehouse: WarehouseRef{ID: wh.ID, Name: wh.Name, Code: wh.Code},
		Summary:   summary,
		Items:     items,
		Pagination: WarehouseInventoryPagination{
			Page:     q.Page,
			PageSize: q.PageSize,
			Total:    total,
		},
	}, nil
}

func (s Service) UpdateProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID, productID string, quantity int) error {
	// Phase W0: the destructive absolute-quantity update deleted stock rows at other
	// locations and wrote no movement. It is disabled — use the inventory stock
	// adjustment (which posts an auditable IN/OUT movement) instead.
	return ErrWarehouseDirectStockDisabled
}

func (s Service) RemoveProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID, productID string) error {

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return err
	}

	// Phase W0 safety: refuse to delete stock rows that still hold quantity — that
	// was silent stock loss with no movement. Removal is only allowed once the
	// product has zero on-hand in this warehouse (transfer/adjust to zero first).
	total, err := s.repo.ProductTotalQtyInWarehouse(ctx, warehouseID, productID)
	if err != nil {
		return err
	}
	if total > 0 {
		return ErrProductHasStock
	}

	return s.repo.RemoveProduct(ctx, storeID, warehouseID, productID)
}

// ──────────────────────────────────────────────
// Transfer stock service methods
// ──────────────────────────────────────────────

// TransferStock — Phase W4A §11: the legacy warehouse-level transfer is DISABLED.
// Its previous behaviour (auto source-location selection, cross-store product cloning,
// auto-created warehouses/locations) is unsafe and has no remaining UI consumer. All
// transfers now go through the canonical, explicit, atomic location→location endpoint
// (stock_movement.TransferStock). The unsafe repository implementation is no longer
// reachable. This guard returns a clear domain error for any residual caller. Historical
// transfer data is untouched.
func (s Service) TransferStock(_ context.Context, _ auth.Claims, _, _ string, _ WarehouseTransferRequest) error {
	return ErrTransferLegacyDisabled
}

// ──────────────────────────────────────────────
// Warehouse Inventory service methods
// ──────────────────────────────────────────────

func (s Service) ListInventory(ctx context.Context, actor auth.Claims, storeID, warehouseID string) ([]WarehouseInventory, error) {

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return nil, err
	}

	return s.repo.ListWarehouseInventory(ctx, warehouseID)
}

func (s Service) AllocateInventory(ctx context.Context, actor auth.Claims, storeID, warehouseID, productID string, req AllocateInventoryRequest) error {
	if req.Quantity <= 0 {
		return ErrAllocateZeroQty
	}

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return err
	}

	// Verify product belongs to store
	if _, err := s.repo.ProductBelongsToStore(ctx, storeID, productID); err != nil {
		return err
	}

	return s.repo.AllocateInventoryToStock(ctx, storeID, warehouseID, productID, req.Quantity, req.Note, actor.UserID)
}
