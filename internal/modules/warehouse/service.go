package warehouse

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct{ repo Repository }

func NewService(repo Repository) Service { return Service{repo: repo} }

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateWarehouseRequest) (Warehouse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return Warehouse{}, ErrInvalidName
	}
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Warehouse{}, err
	}
	if !allowed {
		return Warehouse{}, ErrForbiddenStoreAccess
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

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]Warehouse, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, id string) (Warehouse, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Warehouse{}, err
	}
	if !allowed {
		return Warehouse{}, ErrForbiddenStoreAccess
	}
	return s.repo.GetByID(ctx, storeID, id)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateWarehouseRequest) (Warehouse, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Warehouse{}, err
	}
	if !allowed {
		return Warehouse{}, ErrForbiddenStoreAccess
	}
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

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, id string) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}
	return s.repo.Delete(ctx, storeID, id)
}

// ──────────────────────────────────────────────
// Warehouse-Product service methods
// ──────────────────────────────────────────────

func (s Service) AddProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID string, req AddWarehouseProductRequest) (WarehouseProduct, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return WarehouseProduct{}, err
	}
	if !allowed {
		return WarehouseProduct{}, ErrForbiddenStoreAccess
	}
	// Phase W0: the legacy direct add changed stocks without a movement and could
	// silently auto-create locations. It is disabled — receiving now goes through
	// the canonical Goods Receipt workflow.
	return WarehouseProduct{}, ErrWarehouseDirectStockDisabled
}

func (s Service) ListProducts(ctx context.Context, actor auth.Claims, storeID, warehouseID string) ([]WarehouseProduct, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}

	// Verify warehouse belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return nil, err
	}

	return s.repo.ListProducts(ctx, warehouseID)
}

func (s Service) UpdateProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID, productID string, quantity int) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}
	// Phase W0: the destructive absolute-quantity update deleted stock rows at other
	// locations and wrote no movement. It is disabled — use the inventory stock
	// adjustment (which posts an auditable IN/OUT movement) instead.
	return ErrWarehouseDirectStockDisabled
}

func (s Service) RemoveProduct(ctx context.Context, actor auth.Claims, storeID, warehouseID, productID string) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}

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

func (s Service) TransferStock(ctx context.Context, actor auth.Claims, storeID, warehouseID string, req WarehouseTransferRequest) error {
	// Validate quantity
	if req.Quantity <= 0 {
		return ErrTransferZeroQty
	}

	// Validate destination type
	if req.DestinationType != "warehouse" && req.DestinationType != "stock" {
		return ErrTransferInvalidDestination
	}

	// Verify user can manage this store
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}

	// If destination_store_id is provided, verify user can also manage the target store
	if req.DestinationStoreID != "" {
		allowedDest, err := s.repo.UserCanManageStore(ctx, req.DestinationStoreID, actor.UserID, actor.Role)
		if err != nil {
			return err
		}
		if !allowedDest {
			return ErrForbiddenStoreAccess
		}
	}

	// Verify source warehouse exists and belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, warehouseID); err != nil {
		return err
	}

	// Verify product belongs to store
	if _, err := s.repo.ProductBelongsToStore(ctx, storeID, req.ProductID); err != nil {
		return err
	}

	// Determine destination identifier
	var destID string
	switch req.DestinationType {
	case "warehouse":
		isCrossStore := req.DestinationStoreID != "" && req.DestinationStoreID != storeID
		if isCrossStore {
			// Cross-store: the repository auto-clones the warehouse in the target store.
			// A specific destination_id is optional — if given, it must exist in the target store.
			destID = req.DestinationID
			if destID != "" {
				if destID == warehouseID {
					return ErrTransferSameWarehouse
				}
				if _, err := s.repo.GetByID(ctx, req.DestinationStoreID, destID); err != nil {
					return err
				}
			}
		} else {
			// Same-store: destination warehouse must be provided and must differ from source.
			destID = req.DestinationID
			if destID == "" {
				return ErrTransferInvalidDestination
			}
			if destID == warehouseID {
				return ErrTransferSameWarehouse
			}
			if _, err := s.repo.GetByID(ctx, storeID, destID); err != nil {
				return err
			}
		}
	case "stock":
		destID = "stock"
	}

	return s.repo.TransferStock(ctx, storeID, warehouseID, req.ProductID, req.Quantity, destID, req.DestinationStoreID, req.Note, actor.UserID)
}

// ──────────────────────────────────────────────
// Warehouse Inventory service methods
// ──────────────────────────────────────────────

func (s Service) ListInventory(ctx context.Context, actor auth.Claims, storeID, warehouseID string) ([]WarehouseInventory, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}

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

	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
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
