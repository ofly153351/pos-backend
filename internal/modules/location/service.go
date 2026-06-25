package location

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/lifecycle"
)

type Service struct {
	repo Repository
	db   *gorm.DB
}

// DeleteOutcome is the result of a smart delete: the action that was applied ("archived" or
// "deleted") plus the dependency assessment that drove it (for the client toast/log).
type DeleteOutcome struct {
	Action     string               `json:"action"`
	Assessment lifecycle.Assessment `json:"assessment"`
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

func (s Service) canView(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := s.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ?", storeID, userID).
		Count(&count).Error
	return count > 0, err
}

func (s Service) warehouseBelongsToStore(ctx context.Context, storeID, warehouseID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("warehouses").
		Where("id = ? AND store_id = ?", warehouseID, storeID).
		Count(&count).Error
	return count > 0, err
}

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, input CreateLocationRequest) (Location, error) {
	if strings.TrimSpace(storeID) == "" {
		return Location{}, ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Location{}, err
	}
	if !allowed {
		return Location{}, ErrLocationForbidden
	}
	if strings.TrimSpace(input.Name) == "" {
		return Location{}, ErrLocationNameRequired
	}
	ok, err := s.warehouseBelongsToStore(ctx, storeID, input.WarehouseID)
	if err != nil {
		return Location{}, err
	}
	if !ok {
		return Location{}, ErrInvalidWarehouse
	}
	now := time.Now().UTC()
	loc := Location{
		ID:          newID(),
		StoreID:     storeID,
		WarehouseID: input.WarehouseID,
		Name:        strings.TrimSpace(input.Name),
		Code:        strings.TrimSpace(input.Code),
		ZoneName:    strings.TrimSpace(input.ZoneName),
		FloorName:   strings.TrimSpace(input.FloorName),
		IsSalePoint: input.IsSalePoint,
		IsActive:    true,
		CreatedAt:   now,
	}
	return s.repo.Create(ctx, loc)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string, filter ListFilter) (ListResult, error) {
	if strings.TrimSpace(storeID) == "" {
		return ListResult{}, ErrLocationForbidden
	}
	filter.WarehouseID = strings.TrimSpace(filter.WarehouseID)
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ListResult{}, err
	}
	if !allowed {
		return ListResult{}, ErrLocationForbidden
	}
	if filter.WarehouseID != "" {
		ok, err := s.warehouseBelongsToStore(ctx, storeID, filter.WarehouseID)
		if err != nil {
			return ListResult{}, err
		}
		if !ok {
			return ListResult{}, ErrInvalidWarehouse
		}
	}
	return s.repo.ListByStore(ctx, storeID, filter)
}

func (s Service) ListTree(ctx context.Context, actor auth.Claims, storeID, warehouseID string) ([]TreeZone, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, ErrLocationForbidden
	}
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrLocationForbidden
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID != "" {
		ok, err := s.warehouseBelongsToStore(ctx, storeID, warehouseID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidWarehouse
		}
	}
	return s.repo.GetTree(ctx, storeID, warehouseID)
}

func (s Service) ListProducts(ctx context.Context, actor auth.Claims, storeID, locationID string, page, limit int) ([]LocationProduct, int64, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, 0, ErrLocationForbidden
	}
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, 0, err
	}
	if !allowed {
		return nil, 0, ErrLocationForbidden
	}
	// Confirm location belongs to store
	if _, err := s.repo.GetByID(ctx, storeID, locationID); err != nil {
		return nil, 0, err
	}
	return s.repo.GetProducts(ctx, storeID, locationID, page, limit)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, locationID string) (Location, error) {
	if strings.TrimSpace(storeID) == "" {
		return Location{}, ErrLocationForbidden
	}
	allowed, err := s.canView(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Location{}, err
	}
	if !allowed {
		return Location{}, ErrLocationForbidden
	}
	return s.repo.GetByID(ctx, storeID, locationID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, locationID string, input UpdateLocationRequest) (Location, error) {
	if strings.TrimSpace(storeID) == "" {
		return Location{}, ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Location{}, err
	}
	if !allowed {
		return Location{}, ErrLocationForbidden
	}
	current, err := s.repo.GetByID(ctx, storeID, locationID)
	if err != nil {
		return Location{}, err
	}
	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.Code != nil {
		current.Code = strings.TrimSpace(*input.Code)
	}
	if input.ZoneName != nil {
		current.ZoneName = strings.TrimSpace(*input.ZoneName)
	}
	if input.FloorName != nil {
		current.FloorName = strings.TrimSpace(*input.FloorName)
	}
	if input.IsSalePoint != nil {
		// Phase W1 guard: the default sale location must stay a sale point. Turning it
		// off silently would leave the store without a default POS location, so require
		// choosing a replacement default first.
		if current.IsDefaultSale && !*input.IsSalePoint {
			return Location{}, ErrDefaultSaleLocationDeactivate
		}
		current.IsSalePoint = *input.IsSalePoint
	}
	if input.IsActive != nil {
		// Phase W1 invariant: the default sale location must stay active. Disabling it
		// would leave the store with no usable default POS location, so require choosing
		// a replacement default first.
		if current.IsDefaultSale && current.IsActive && !*input.IsActive {
			return Location{}, ErrDefaultSaleLocationDisable
		}
		current.IsActive = *input.IsActive
	}
	current.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, current)
}

// AssessDeletion returns the read-only deletion assessment for a location (drives the
// adaptive delete/archive modal). Manage-level access required.
func (s Service) AssessDeletion(ctx context.Context, actor auth.Claims, storeID, locationID string) (lifecycle.Assessment, error) {
	if strings.TrimSpace(storeID) == "" {
		return lifecycle.Assessment{}, ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return lifecycle.Assessment{}, err
	}
	if !allowed {
		return lifecycle.Assessment{}, ErrLocationForbidden
	}
	// Confirm the location exists and is in this store (→ 404 otherwise).
	if _, err := s.repo.GetByID(ctx, storeID, locationID); err != nil {
		return lifecycle.Assessment{}, err
	}
	b, err := s.repo.GatherDeletionBlockers(ctx, storeID, locationID)
	if err != nil {
		return lifecycle.Assessment{}, err
	}
	return lifecycle.Assess(lifecycle.EntityLocation, b), nil
}

// Delete is the smart delete (§6): it resolves the safe strategy and applies it atomically.
// A never-used location is hard-deleted; a used location with zero stock and history is
// archived (soft-deleted); anything with a hard blocker returns the structured blocker error
// with no mutation. expected is an optional client-declared action for optimistic
// concurrency (LOCATION/ENTITY_STATE_CHANGED on mismatch).
func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, locationID, expected string) (DeleteOutcome, error) {
	if strings.TrimSpace(storeID) == "" {
		return DeleteOutcome{}, ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return DeleteOutcome{}, err
	}
	if !allowed {
		return DeleteOutcome{}, ErrLocationForbidden
	}
	// Confirm existence + store scope before the locked apply (→ 404 otherwise).
	if _, err := s.repo.GetByID(ctx, storeID, locationID); err != nil {
		return DeleteOutcome{}, err
	}
	assessment, applied, err := s.repo.ApplyDeletion(ctx, storeID, locationID, expected)
	if err != nil {
		return DeleteOutcome{Assessment: assessment}, err
	}
	return DeleteOutcome{Action: applied, Assessment: assessment}, nil
}

func (s Service) RenameZone(ctx context.Context, actor auth.Claims, storeID, warehouseID, oldZone, newZone string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrLocationForbidden
	}
	if warehouseID != "" {
		if ok, err := s.warehouseBelongsToStore(ctx, storeID, warehouseID); err != nil {
			return err
		} else if !ok {
			return ErrInvalidWarehouse
		}
	}
	_, err = s.repo.RenameZone(ctx, storeID, warehouseID, oldZone, newZone)
	return err
}

func (s Service) DeleteZone(ctx context.Context, actor auth.Claims, storeID, warehouseID, zoneName string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrLocationForbidden
	}
	if warehouseID != "" {
		if ok, err := s.warehouseBelongsToStore(ctx, storeID, warehouseID); err != nil {
			return err
		} else if !ok {
			return ErrInvalidWarehouse
		}
	}
	return s.repo.DeleteZone(ctx, storeID, warehouseID, zoneName)
}

func (s Service) RenameFloor(ctx context.Context, actor auth.Claims, storeID, warehouseID, zoneName, oldFloor, newFloor string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrLocationForbidden
	}
	if warehouseID != "" {
		if ok, err := s.warehouseBelongsToStore(ctx, storeID, warehouseID); err != nil {
			return err
		} else if !ok {
			return ErrInvalidWarehouse
		}
	}
	_, err = s.repo.RenameFloor(ctx, storeID, warehouseID, zoneName, oldFloor, newFloor)
	return err
}

func (s Service) DeleteFloor(ctx context.Context, actor auth.Claims, storeID, warehouseID, zoneName, floorName string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrLocationForbidden
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrLocationForbidden
	}
	if warehouseID != "" {
		if ok, err := s.warehouseBelongsToStore(ctx, storeID, warehouseID); err != nil {
			return err
		} else if !ok {
			return ErrInvalidWarehouse
		}
	}
	return s.repo.DeleteFloor(ctx, storeID, warehouseID, zoneName, floorName)
}
