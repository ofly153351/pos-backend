package location

import "time"

type Location struct {
	ID            string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID       string    `json:"store_id" gorm:"column:store_id"`
	WarehouseID   string    `json:"warehouse_id" gorm:"column:warehouse_id"`
	Name          string    `json:"name" gorm:"column:name"`
	Code          string    `json:"code,omitempty" gorm:"column:code"`
	ZoneName      string    `json:"zone_name,omitempty" gorm:"column:zone_name"`
	FloorName     string    `json:"floor_name,omitempty" gorm:"column:floor_name"`
	IsSalePoint   bool      `json:"is_sale_point" gorm:"column:is_sale_point"`
	IsDefaultSale bool      `json:"is_default_sale" gorm:"column:is_default_sale"`
	IsActive      bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at"`
	// DeletedAt: see warehouse.Warehouse.DeletedAt — a plain *time.Time (NOT gorm.DeletedAt)
	// so GORM never auto-filters; archived rows expose it so the FE can render the Archived
	// state and badge. omitempty keeps live-row responses byte-identical to before.
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"column:deleted_at"`

	// Relations
	WarehouseName string `json:"warehouse_name,omitempty" gorm:"-"`
}

func (Location) TableName() string { return "locations" }

type CreateLocationRequest struct {
	WarehouseID string `json:"warehouse_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	ZoneName    string `json:"zone_name"`
	FloorName   string `json:"floor_name"`
	IsSalePoint bool   `json:"is_sale_point"`
}

type UpdateLocationRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	ZoneName    *string `json:"zone_name"`
	FloorName   *string `json:"floor_name"`
	IsSalePoint *bool   `json:"is_sale_point"`
	IsActive    *bool   `json:"is_active"`
}

// ListFilter holds optional query filters for listing locations.
type ListFilter struct {
	WarehouseID string
	ZoneName    string
	FloorName   string
	Search      string
	Page        int
	Limit       int
	// IncludeArchived returns soft-deleted (archived) locations as well. Off by default so
	// management lists and operational selectors only see live locations (migration 047).
	IncludeArchived bool
}

// ListResult is the paginated response for location lists.
type ListResult struct {
	Items      []Location `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}

// TreeFloor is a floor/shelf node in the warehouse tree.
type TreeFloor struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// TreeZone is a zone node containing floors in the warehouse tree.
type TreeZone struct {
	Name   string      `json:"name"`
	Count  int64       `json:"count"`
	Floors []TreeFloor `json:"floors"`
}

// LocationProduct is a product stored at a location, with the master-data
// fields a stock-count sheet needs (barcode / min_stock / cost basis /
// category) so clients never have to join the whole catalog client-side.
// Quantity is stocks.quantity at this location; rows exist only when > 0.
type LocationProduct struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	SKU         string  `json:"sku,omitempty"`
	Barcode     string  `json:"barcode,omitempty"`
	Quantity    int     `json:"quantity"`
	MinStock    int     `json:"min_stock"`
	CostPrice   float64 `json:"cost_price"`
	BasePrice   float64 `json:"base_price"`
	CategoryID  string  `json:"category_id,omitempty"`
}
