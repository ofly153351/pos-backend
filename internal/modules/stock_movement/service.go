package stock_movement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
	db   *gorm.DB
}

func NewService(repo Repository, db *gorm.DB) Service {
	return Service{repo: repo, db: db}
}

func (s Service) canManage(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := s.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager"}).
		Count(&count).Error
	return count > 0, err
}

func (s Service) productExistsInStore(ctx context.Context, storeID, productID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("products").
		Where("id = ? AND store_id = ?", productID, storeID).
		Count(&count).Error
	return count > 0, err
}

// findDefaultStockLocation returns the first active sale-point location for the store.
// Falls back to any active location so stock is always recorded in the stocks table.
func (s Service) findDefaultStockLocation(ctx context.Context, storeID string) (string, error) {
	var locID string
	err := s.db.WithContext(ctx).
		Table("locations").
		Select("id").
		Where("store_id = ? AND is_sale_point = TRUE AND is_active = TRUE", storeID).
		Order("created_at ASC").
		Take(&locID).Error
	if err == nil {
		return locID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	// Fallback: any active location in the store
	err = s.db.WithContext(ctx).
		Table("locations").
		Select("id").
		Where("store_id = ? AND is_active = TRUE", storeID).
		Order("created_at ASC").
		Take(&locID).Error
	if err == nil {
		return locID, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return "", err
}

// AddStock creates IN movements and adds stock to locations
func (s Service) AddStock(ctx context.Context, actor auth.Claims, storeID string, input AddStockRequest) (AdditionResult, error) {
	if strings.TrimSpace(storeID) == "" {
		return AdditionResult{}, fmt.Errorf("storeID is required")
	}

	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return AdditionResult{}, err
	}
	if !allowed {
		return AdditionResult{}, ErrStockForbidden
	}

	if len(input.Items) == 0 {
		return AdditionResult{}, ErrStockNoItems
	}

	for _, item := range input.Items {
		if item.Quantity <= 0 {
			return AdditionResult{}, ErrStockBadQty
		}
	}

	now := time.Now().UTC()
	var movements []StockMovement

	for _, item := range input.Items {
		exists, err := s.productExistsInStore(ctx, storeID, item.ProductID)
		if err != nil {
			return AdditionResult{}, err
		}
		if !exists {
			return AdditionResult{}, ErrProductNotFound
		}

		// Resolve the target location: use the provided one, or auto-find the
		// store's first active sale-point location so the stocks table is always updated.
		resolvedLocID := item.LocationID
		if resolvedLocID == "" {
			defaultLoc, err := s.findDefaultStockLocation(ctx, storeID)
			if err != nil {
				return AdditionResult{}, err
			}
			resolvedLocID = defaultLoc
		}

		var locPtr *string
		if resolvedLocID != "" {
			locPtr = &resolvedLocID
		}

		mg := StockMovement{
			ID:             newID(),
			StoreID:        storeID,
			ProductID:      item.ProductID,
			LocationID:     locPtr,
			QuantityChange: item.Quantity,
			Type:           MovementTypeIn,
			Note:           strings.TrimSpace(item.Note),
			CreatedBy:      actor.UserID,
			CreatedAt:      now,
		}

		created, err := s.repo.Create(ctx, mg)
		if err != nil {
			return AdditionResult{}, err
		}

		// Update stock in the locations table
		if resolvedLocID != "" {
			if err := s.repo.UpsertStock(ctx, storeID, item.ProductID, resolvedLocID, item.Quantity); err != nil {
				return AdditionResult{}, err
			}
		}

		movements = append(movements, created)
	}

	return AdditionResult{Movements: movements}, nil
}

// RemoveStock creates OUT movements and deducts stock from locations
func (s Service) RemoveStock(ctx context.Context, actor auth.Claims, storeID string, req RemoveStockRequest) (StockMovement, error) {
	if strings.TrimSpace(storeID) == "" {
		return StockMovement{}, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return StockMovement{}, err
	}
	if !allowed {
		return StockMovement{}, ErrStockForbidden
	}
	if req.Quantity <= 0 {
		return StockMovement{}, ErrStockBadQty
	}

	exists, err := s.productExistsInStore(ctx, storeID, req.ProductID)
	if err != nil {
		return StockMovement{}, err
	}
	if !exists {
		return StockMovement{}, ErrProductNotFound
	}

	now := time.Now().UTC()
	var locID *string
	if req.LocationID != "" {
		locID = &req.LocationID
	}

	mg := StockMovement{
		ID:             newID(),
		StoreID:        storeID,
		ProductID:      req.ProductID,
		LocationID:     locID,
		QuantityChange: -req.Quantity,
		Type:           MovementTypeOut,
		Note:           strings.TrimSpace(req.Note),
		CreatedBy:      actor.UserID,
		CreatedAt:      now,
	}

	created, err := s.repo.Create(ctx, mg)
	if err != nil {
		return StockMovement{}, err
	}

	if req.LocationID != "" {
		if err := s.repo.UpsertStock(ctx, storeID, req.ProductID, req.LocationID, -req.Quantity); err != nil {
			return StockMovement{}, err
		}
	}

	return created, nil
}

// TransferStock moves stock between locations
func (s Service) TransferStock(ctx context.Context, actor auth.Claims, storeID string, req TransferStockRequest) ([]StockMovement, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrStockForbidden
	}
	if req.Quantity <= 0 {
		return nil, ErrStockBadQty
	}
	if req.SourceLocationID == req.DestLocationID {
		return nil, ErrLocationMismatch
	}

	exists, err := s.productExistsInStore(ctx, storeID, req.ProductID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProductNotFound
	}

	now := time.Now().UTC()
	var movements []StockMovement

	// Create OUT movement from source
	outMg := StockMovement{
		ID:                    newID(),
		StoreID:               storeID,
		ProductID:             req.ProductID,
		LocationID:            &req.SourceLocationID,
		DestinationLocationID: &req.DestLocationID,
		QuantityChange:        -req.Quantity,
		Type:                  MovementTypeTransfer,
		Note:                  strings.TrimSpace(req.Note),
		CreatedBy:             actor.UserID,
		CreatedAt:             now,
	}

	outCreated, err := s.repo.Create(ctx, outMg)
	if err != nil {
		return nil, err
	}

	// Deduct from source
	if err := s.repo.UpsertStock(ctx, storeID, req.ProductID, req.SourceLocationID, -req.Quantity); err != nil {
		return nil, err
	}

	// Add to destination
	if err := s.repo.UpsertStock(ctx, storeID, req.ProductID, req.DestLocationID, req.Quantity); err != nil {
		return nil, err
	}

	movements = append(movements, outCreated)
	return movements, nil
}

// AdjustStock sets physical stock count at a location, creates ADJUST movement
func (s Service) AdjustStock(ctx context.Context, actor auth.Claims, storeID string, req AdjustStockRequest) (StockMovement, error) {
	if strings.TrimSpace(storeID) == "" {
		return StockMovement{}, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return StockMovement{}, err
	}
	if !allowed {
		return StockMovement{}, ErrStockForbidden
	}

	exists, err := s.productExistsInStore(ctx, storeID, req.ProductID)
	if err != nil {
		return StockMovement{}, err
	}
	if !exists {
		return StockMovement{}, ErrProductNotFound
	}

	// Resolve location — use provided one or fall back to store's default sale-point location
	resolvedLocID := req.LocationID
	if resolvedLocID == "" {
		defaultLoc, err := s.findDefaultStockLocation(ctx, storeID)
		if err != nil {
			return StockMovement{}, err
		}
		resolvedLocID = defaultLoc
	}

	var locPtr *string
	if resolvedLocID != "" {
		locPtr = &resolvedLocID
	}

	// Get current quantity at resolved location
	currentQty, err := s.repo.GetCurrentStockQty(ctx, storeID, req.ProductID, resolvedLocID)
	if err != nil {
		return StockMovement{}, err
	}

	diff := req.PhysicalQty - currentQty
	now := time.Now().UTC()
	movementType := strings.TrimSpace(req.MovementType)
	if movementType == "" {
		movementType = MovementTypeAdjust
	}
	var refPtr *string
	if ref := strings.TrimSpace(req.ReferenceID); ref != "" {
		refPtr = &ref
	}

	mg := StockMovement{
		ID:             newID(),
		StoreID:        storeID,
		ProductID:      req.ProductID,
		LocationID:     locPtr,
		QuantityChange: diff,
		Type:           movementType,
		ReferenceID:    refPtr,
		Note:           fmt.Sprintf("adjusted from %d to %d. %s", currentQty, req.PhysicalQty, strings.TrimSpace(req.Note)),
		CreatedBy:      actor.UserID,
		CreatedAt:      now,
	}

	created, err := s.repo.Create(ctx, mg)
	if err != nil {
		return StockMovement{}, err
	}

	if err := s.repo.SetStockQuantity(ctx, storeID, req.ProductID, resolvedLocID, req.PhysicalQty); err != nil {
		return StockMovement{}, err
	}

	return created, nil
}

// RecordSaleMovement creates a SALE movement and deducts stock
func (s Service) RecordSaleMovement(ctx context.Context, actor auth.Claims, storeID, productID, locationID string, qty int, referenceID string) error {
	if qty <= 0 {
		return ErrStockBadQty
	}

	now := time.Now().UTC()
	mg := StockMovement{
		ID:             newID(),
		StoreID:        storeID,
		ProductID:      productID,
		LocationID:     &locationID,
		QuantityChange: -qty,
		Type:           MovementTypeSale,
		ReferenceID:    &referenceID,
		Note:           "sale deduction",
		CreatedBy:      actor.UserID,
		CreatedAt:      now,
	}

	if _, err := s.repo.Create(ctx, mg); err != nil {
		return err
	}

	if err := s.repo.UpsertStock(ctx, storeID, productID, locationID, -qty); err != nil {
		return err
	}

	return nil
}

// ListMovements lists stock movements for a store
func (s Service) ListMovements(ctx context.Context, actor auth.Claims, storeID string, q ListMovementsQuery) (MovementResponse, error) {
	if strings.TrimSpace(storeID) == "" {
		return MovementResponse{}, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return MovementResponse{}, err
	}
	if !allowed {
		return MovementResponse{}, ErrStockForbidden
	}
	return s.repo.ListByStore(ctx, storeID, q)
}
