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
	ListByStore(ctx context.Context, storeID string, filter ListFilter) (ListResult, error)
	GetByID(ctx context.Context, storeID, locationID string) (Location, error)
	Update(ctx context.Context, loc Location) (Location, error)
	Delete(ctx context.Context, storeID, locationID string) error
	GetTree(ctx context.Context, storeID, warehouseID string) ([]TreeZone, error)
	GetProducts(ctx context.Context, storeID, locationID string, page, limit int) ([]LocationProduct, int64, error)
	RenameZone(ctx context.Context, storeID, warehouseID, oldZone, newZone string) (int64, error)
	DeleteZone(ctx context.Context, storeID, warehouseID, zoneName string) error
	RenameFloor(ctx context.Context, storeID, warehouseID, zoneName, oldFloor, newFloor string) (int64, error)
	DeleteFloor(ctx context.Context, storeID, warehouseID, zoneName, floorName string) error
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

func (r PostgresRepository) buildListQuery(storeID string, filter ListFilter) *gorm.DB {
	q := r.locationBaseQuery().Where("locations.store_id = ?", storeID)
	if filter.WarehouseID != "" {
		q = q.Where("locations.warehouse_id = ?", filter.WarehouseID)
	}
	if filter.ZoneName != "" {
		q = q.Where("COALESCE(locations.zone_name, '') = ?", filter.ZoneName)
	}
	if filter.FloorName != "" {
		q = q.Where("COALESCE(locations.floor_name, '') = ?", filter.FloorName)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("(locations.code ILIKE ? OR locations.name ILIKE ?)", like, like)
	}
	return q
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string, filter ListFilter) (ListResult, error) {
	q := r.buildListQuery(storeID, filter)

	// Count using a separate query on the same table/joins/conditions to avoid subquery issues
	var total int64
	countQ := r.db.WithContext(ctx).
		Table("locations").
		Joins("LEFT JOIN warehouses ON warehouses.id = locations.warehouse_id").
		Where("locations.store_id = ?", storeID)
	if filter.WarehouseID != "" {
		countQ = countQ.Where("locations.warehouse_id = ?", filter.WarehouseID)
	}
	if filter.ZoneName != "" {
		countQ = countQ.Where("COALESCE(locations.zone_name, '') = ?", filter.ZoneName)
	}
	if filter.FloorName != "" {
		countQ = countQ.Where("COALESCE(locations.floor_name, '') = ?", filter.FloorName)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		countQ = countQ.Where("(locations.code ILIKE ? OR locations.name ILIKE ?)", like, like)
	}
	if err := countQ.Count(&total).Error; err != nil {
		return ListResult{}, err
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit > 0 {
		q = q.Limit(limit).Offset((page - 1) * limit)
	}

	var rows []locationQueryRow
	if err := q.WithContext(ctx).
		Order("locations.zone_name ASC NULLS FIRST, locations.floor_name ASC NULLS FIRST, locations.code ASC, locations.name ASC").
		Find(&rows).Error; err != nil {
		return ListResult{}, err
	}

	items := make([]Location, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toLocation())
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return ListResult{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
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
	// Phase W0 safety: stocks.location_id is ON DELETE CASCADE, so a raw delete
	// silently destroys stock rows (and stock_movements.location_id would be nulled).
	// Explicitly reject when the location is referenced or in use.
	inUse, err := r.locationInUse(ctx, locationID)
	if err != nil {
		return err
	}
	if inUse {
		return ErrLocationInUse
	}
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

// locationInUse reports whether a location still holds stock or is referenced by
// operational data — any of which a delete would destroy or orphan.
func (r PostgresRepository) locationInUse(ctx context.Context, locationID string) (bool, error) {
	var n int64
	// Any stock row (even zero quantity — the slot still maps stock here).
	if err := r.db.WithContext(ctx).Table("stocks").Where("location_id = ?", locationID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	// Any movement history (as source or transfer destination).
	if err := r.db.WithContext(ctx).Table("stock_movements").
		Where("location_id = ? OR destination_location_id = ?", locationID, locationID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	// Referenced as a product's default receiving location.
	if err := r.db.WithContext(ctx).Table("products").Where("default_location_id = ?", locationID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	// Still configured as a sale point (POS depends on it).
	if err := r.db.WithContext(ctx).Table("locations").
		Where("id = ? AND is_sale_point = ?", locationID, true).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	// Referenced by a goods-receipt line.
	if err := r.db.WithContext(ctx).Table("warehouse_receipt_items").Where("location_id = ?", locationID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

type treeRow struct {
	ZoneName  *string `gorm:"column:zone_name"`
	FloorName *string `gorm:"column:floor_name"`
	Count     int64   `gorm:"column:count"`
}

func (r PostgresRepository) GetTree(ctx context.Context, storeID, warehouseID string) ([]TreeZone, error) {
	q := r.db.WithContext(ctx).
		Table("locations").
		Select("zone_name, floor_name, COUNT(*) AS count").
		Where("store_id = ?", storeID)
	if warehouseID != "" {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	var rows []treeRow
	if err := q.Group("zone_name, floor_name").
		Order("zone_name ASC NULLS FIRST, floor_name ASC NULLS FIRST").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	// Aggregate rows into zone → floor hierarchy
	zoneIndex := map[string]int{}
	var zones []TreeZone
	for _, row := range rows {
		zoneName := ""
		if row.ZoneName != nil {
			zoneName = *row.ZoneName
		}
		floorName := ""
		if row.FloorName != nil {
			floorName = *row.FloorName
		}
		idx, exists := zoneIndex[zoneName]
		if !exists {
			idx = len(zones)
			zoneIndex[zoneName] = idx
			zones = append(zones, TreeZone{Name: zoneName, Floors: []TreeFloor{}})
		}
		zones[idx].Count += row.Count
		zones[idx].Floors = append(zones[idx].Floors, TreeFloor{Name: floorName, Count: row.Count})
	}
	return zones, nil
}

type productAtLocationRow struct {
	ProductID   string `gorm:"column:product_id"`
	ProductName string `gorm:"column:product_name"`
	SKU         string `gorm:"column:sku"`
	Quantity    int    `gorm:"column:quantity"`
}

func (r PostgresRepository) GetProducts(ctx context.Context, storeID, locationID string, page, limit int) ([]LocationProduct, int64, error) {
	q := r.db.WithContext(ctx).
		Table("stocks").
		Select("stocks.product_id, products.name AS product_name, COALESCE(products.sku, '') AS sku, stocks.quantity").
		Joins("LEFT JOIN products ON products.id = stocks.product_id").
		Where("stocks.store_id = ? AND stocks.location_id = ? AND stocks.quantity > 0", storeID, locationID)

	var total int64
	if err := r.db.WithContext(ctx).Table("(?) AS sub", q).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if limit > 0 {
		q = q.Limit(limit).Offset((page - 1) * limit)
	}

	var rows []productAtLocationRow
	if err := q.Order("products.name ASC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	products := make([]LocationProduct, 0, len(rows))
	for _, row := range rows {
		products = append(products, LocationProduct{
			ProductID:   row.ProductID,
			ProductName: row.ProductName,
			SKU:         row.SKU,
			Quantity:    row.Quantity,
		})
	}
	return products, total, nil
}

func (r PostgresRepository) zoneWhere(q *gorm.DB, zoneName string) *gorm.DB {
	if zoneName == "" {
		return q.Where("zone_name IS NULL")
	}
	return q.Where("zone_name = ?", zoneName)
}

func (r PostgresRepository) floorWhere(q *gorm.DB, floorName string) *gorm.DB {
	if floorName == "" {
		return q.Where("floor_name IS NULL")
	}
	return q.Where("floor_name = ?", floorName)
}

func (r PostgresRepository) warehouseWhere(q *gorm.DB, warehouseID string) *gorm.DB {
	if warehouseID == "" {
		return q
	}
	return q.Where("warehouse_id = ?", warehouseID)
}

func (r PostgresRepository) locationIDsInZone(ctx context.Context, storeID, warehouseID, zoneName string) ([]string, error) {
	q := r.db.WithContext(ctx).Table("locations").Where("store_id = ?", storeID)
	q = r.warehouseWhere(q, warehouseID)
	q = r.zoneWhere(q, zoneName)
	var ids []string
	return ids, q.Pluck("id", &ids).Error
}

func (r PostgresRepository) locationIDsInFloor(ctx context.Context, storeID, warehouseID, zoneName, floorName string) ([]string, error) {
	q := r.db.WithContext(ctx).Table("locations").Where("store_id = ?", storeID)
	q = r.warehouseWhere(q, warehouseID)
	q = r.zoneWhere(q, zoneName)
	q = r.floorWhere(q, floorName)
	var ids []string
	return ids, q.Pluck("id", &ids).Error
}

func (r PostgresRepository) hasStock(ctx context.Context, ids []string) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("stocks").
		Where("location_id IN ? AND quantity > 0", ids).
		Count(&count).Error
	return count > 0, err
}

func (r PostgresRepository) RenameZone(ctx context.Context, storeID, warehouseID, oldZone, newZone string) (int64, error) {
	q := r.db.WithContext(ctx).Table("locations").Where("store_id = ?", storeID)
	q = r.warehouseWhere(q, warehouseID)
	q = r.zoneWhere(q, oldZone)
	var newVal any
	if strings.TrimSpace(newZone) == "" {
		newVal = nil
	} else {
		newVal = strings.TrimSpace(newZone)
	}
	result := q.Updates(map[string]any{"zone_name": newVal, "updated_at": time.Now().UTC()})
	return result.RowsAffected, result.Error
}

func (r PostgresRepository) DeleteZone(ctx context.Context, storeID, warehouseID, zoneName string) error {
	ids, err := r.locationIDsInZone(ctx, storeID, warehouseID, zoneName)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if stocked, err := r.hasStock(ctx, ids); err != nil {
		return err
	} else if stocked {
		return ErrLocationInUse
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&Location{}).Error
}

func (r PostgresRepository) RenameFloor(ctx context.Context, storeID, warehouseID, zoneName, oldFloor, newFloor string) (int64, error) {
	q := r.db.WithContext(ctx).Table("locations").Where("store_id = ?", storeID)
	q = r.warehouseWhere(q, warehouseID)
	q = r.zoneWhere(q, zoneName)
	q = r.floorWhere(q, oldFloor)
	var newVal any
	if strings.TrimSpace(newFloor) == "" {
		newVal = nil
	} else {
		newVal = strings.TrimSpace(newFloor)
	}
	result := q.Updates(map[string]any{"floor_name": newVal, "updated_at": time.Now().UTC()})
	return result.RowsAffected, result.Error
}

func (r PostgresRepository) DeleteFloor(ctx context.Context, storeID, warehouseID, zoneName, floorName string) error {
	ids, err := r.locationIDsInFloor(ctx, storeID, warehouseID, zoneName, floorName)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if stocked, err := r.hasStock(ctx, ids); err != nil {
		return err
	} else if stocked {
		return ErrLocationInUse
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&Location{}).Error
}
