package location

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, loc Location) (Location, error)
	ListByStore(ctx context.Context, storeID string, warehouseID string) ([]Location, error)
	GetByID(ctx context.Context, storeID, locationID string) (Location, error)
	Update(ctx context.Context, loc Location) (Location, error)
	Delete(ctx context.Context, storeID, locationID string) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

type locationQueryRow struct {
	ID            string    `gorm:"column:id"`
	StoreID       string    `gorm:"column:store_id"`
	WarehouseID   string    `gorm:"column:warehouse_id"`
	Name          string    `gorm:"column:name"`
	Code          *string   `gorm:"column:code"`
	ZoneName      *string   `gorm:"column:zone_name"`
	FloorName     *string   `gorm:"column:floor_name"`
	IsSalePoint   bool      `gorm:"column:is_sale_point"`
	IsActive      bool      `gorm:"column:is_active"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
	WarehouseName string    `gorm:"column:warehouse_name"`
}

func (r *locationQueryRow) toLocation() Location {
	loc := Location{
		ID:            r.ID,
		StoreID:       r.StoreID,
		WarehouseID:   r.WarehouseID,
		Name:          r.Name,
		IsSalePoint:   r.IsSalePoint,
		IsActive:      r.IsActive,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		WarehouseName: r.WarehouseName,
	}
	if r.Code != nil {
		loc.Code = *r.Code
	}
	if r.ZoneName != nil {
		loc.ZoneName = *r.ZoneName
	}
	if r.FloorName != nil {
		loc.FloorName = *r.FloorName
	}
	return loc
}

func (r PostgresRepository) locationBaseQuery() *gorm.DB {
	return r.db.Table("locations").
		Select(`
			locations.id, locations.store_id, locations.warehouse_id,
			locations.name, locations.code, locations.zone_name, locations.floor_name,
			locations.is_sale_point, locations.is_active,
			locations.created_at, locations.updated_at,
			COALESCE(warehouses.name, '') AS warehouse_name
		`).
		Joins("LEFT JOIN warehouses ON warehouses.id = locations.warehouse_id")
}

func (r PostgresRepository) Create(ctx context.Context, loc Location) (Location, error) {
	payload := map[string]any{
		"id":            loc.ID,
		"store_id":      loc.StoreID,
		"warehouse_id":  loc.WarehouseID,
		"name":          loc.Name,
		"is_sale_point": loc.IsSalePoint,
		"is_active":     true,
		"created_at":    loc.CreatedAt,
		"updated_at":    loc.CreatedAt,
	}
	if loc.Code == "" {
		payload["code"] = nil
	} else {
		payload["code"] = loc.Code
	}
	if strings.TrimSpace(loc.ZoneName) == "" {
		payload["zone_name"] = nil
	} else {
		payload["zone_name"] = strings.TrimSpace(loc.ZoneName)
	}
	if strings.TrimSpace(loc.FloorName) == "" {
		payload["floor_name"] = nil
	} else {
		payload["floor_name"] = strings.TrimSpace(loc.FloorName)
	}
	if err := r.db.WithContext(ctx).Table("locations").Create(payload).Error; err != nil {
		if strings.Contains(err.Error(), "SQLSTATE 23505") {
			return Location{}, ErrLocationExists
		}
		return Location{}, err
	}
	return r.GetByID(ctx, loc.StoreID, loc.ID)
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string, warehouseID string) ([]Location, error) {
	var rows []locationQueryRow
	q := r.locationBaseQuery().Where("locations.store_id = ?", storeID)
	if warehouseID != "" {
		q = q.Where("locations.warehouse_id = ?", warehouseID)
	}
	if err := q.Order("locations.name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	locations := make([]Location, 0, len(rows))
	for _, row := range rows {
		locations = append(locations, row.toLocation())
	}
	return locations, nil
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, locationID string) (Location, error) {
	var row locationQueryRow
	err := r.locationBaseQuery().
		Where("locations.store_id = ? AND locations.id = ?", storeID, locationID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Location{}, ErrLocationNotFound
		}
		return Location{}, err
	}
	return row.toLocation(), nil
}

func (r PostgresRepository) Update(ctx context.Context, loc Location) (Location, error) {
	updates := map[string]any{
		"name":          loc.Name,
		"is_sale_point": loc.IsSalePoint,
		"is_active":     loc.IsActive,
		"updated_at":    loc.UpdatedAt,
	}
	if loc.Code == "" {
		updates["code"] = nil
	} else {
		updates["code"] = loc.Code
	}
	if strings.TrimSpace(loc.ZoneName) == "" {
		updates["zone_name"] = nil
	} else {
		updates["zone_name"] = strings.TrimSpace(loc.ZoneName)
	}
	if strings.TrimSpace(loc.FloorName) == "" {
		updates["floor_name"] = nil
	} else {
		updates["floor_name"] = strings.TrimSpace(loc.FloorName)
	}

	result := r.db.WithContext(ctx).
		Table("locations").
		Where("store_id = ? AND id = ?", loc.StoreID, loc.ID).
		Updates(updates)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "SQLSTATE 23505") {
			return Location{}, ErrLocationExists
		}
		return Location{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Location{}, ErrLocationNotFound
	}
	return r.GetByID(ctx, loc.StoreID, loc.ID)
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, locationID string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, locationID).
		Delete(&Location{})
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "SQLSTATE 23503") {
			return ErrLocationInUse
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLocationNotFound
	}
	return nil
}
