package product

import (
	"mime/multipart"
	"time"
)

type Product struct {
	ID                  string     `json:"id" gorm:"column:id;primaryKey"`
	StoreID             string     `json:"store_id" gorm:"column:store_id"`
	StockStatus         string     `json:"stock_status,omitempty" gorm:"column:stock_status"`
	ProductTypeID       string     `json:"product_type_id,omitempty" gorm:"column:product_type_id"`
	ProductTypeName     string     `json:"product_type_name,omitempty" gorm:"-"`
	Name                string     `json:"name" gorm:"column:name"`
	BrandID             string     `json:"brand_id,omitempty" gorm:"column:brand_id"`
	BrandName           string     `json:"brand_name,omitempty" gorm:"-"`
	SKU                 string     `json:"sku,omitempty" gorm:"column:sku"`
	Barcode             string     `json:"barcode,omitempty" gorm:"column:barcode"`
	ProductUnitID       string     `json:"product_unit_id" gorm:"column:product_unit_id"`
	ProductUnitName     string     `json:"product_unit_name,omitempty" gorm:"-"`
	CostPrice           float64    `json:"cost_price,omitempty" gorm:"column:cost_price"`
	ImageURL            string     `json:"image_url,omitempty" gorm:"column:image_url"`
	MinStock            int        `json:"min_stock" gorm:"column:min_stock"`
	MaxStock            *int       `json:"max_stock,omitempty" gorm:"column:max_stock"`
	BasePrice           float64    `json:"base_price" gorm:"column:base_price"`
	SpecialPrice        *float64   `json:"special_price,omitempty" gorm:"column:special_price"`
	SpecialPriceStartAt *time.Time `json:"special_price_start_at,omitempty" gorm:"column:special_price_start_at"`
	SpecialPriceEndAt   *time.Time `json:"special_price_end_at,omitempty" gorm:"column:special_price_end_at"`
	EffectivePrice      float64    `json:"effective_price" gorm:"-"`
	ProductCode         string     `json:"product_code,omitempty" gorm:"column:product_code"`
	Description         string     `json:"description,omitempty" gorm:"column:description"`
	StorageLocation     string     `json:"storage_location,omitempty" gorm:"column:storage_location"`
	DefaultLocationID   *string    `json:"default_location_id" gorm:"column:default_location_id"`
	// Phase W5 stock aggregate contract. total_stock/warehouse_stock are DEPRECATED
	// (historically mislabeled): total_stock == ready_stock (sale-point), warehouse_stock
	// == the grand total. Prefer the explicit fields below. ready_stock + storage_stock
	// == warehouse_stock (grand total). All include inactive-location stock.
	TotalStock     int       `json:"total_stock" gorm:"column:total_stock"`         // DEPRECATED: == ready_stock (sale-point sum)
	WarehouseStock int       `json:"warehouse_stock" gorm:"column:warehouse_stock"` // DEPRECATED: grand total across all locations
	ReadyStock     int       `json:"ready_stock" gorm:"column:ready_stock"`         // W5: SUM where is_sale_point (POS-sellable)
	StorageStock   int       `json:"storage_stock" gorm:"column:storage_stock"`     // W5: SUM where NOT is_sale_point (storage)
	IsActive       bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (Product) TableName() string {
	return "products"
}

type CreateProductRequest struct {
	Name                string
	BrandID             string
	SKU                 string
	Barcode             string
	ProductCode         string
	Description         string
	StorageLocation     string
	DefaultLocationID   string
	ProductTypeID       string
	ProductUnitID       string
	MinStock            *int
	MaxStock            *int
	InitialStock        int
	BasePrice           float64
	CostPrice           float64
	SpecialPrice        *float64
	SpecialPriceStartAt *time.Time
	SpecialPriceEndAt   *time.Time
	IsActive            *bool
	ImageFile           *multipart.FileHeader
}

type UpdateProductRequest struct {
	Name                 *string
	BrandID              *string
	SKU                  *string
	Barcode              *string
	ProductCode          *string
	Description          *string
	StorageLocation      *string
	DefaultLocationID    *string
	ClearBarcode         bool
	ClearSKU             bool
	ClearProductCode     bool
	ClearDefaultLocation bool
	ProductTypeID        *string
	ProductUnitID        *string
	MinStock             *int
	MaxStock             *int
	ClearMaxStock        bool
	BasePrice            *float64
	CostPrice            *float64
	SpecialPrice         *float64
	ClearSpecialPrice    bool
	SpecialPriceStartAt  *time.Time
	SpecialPriceEndAt    *time.Time
	ClearSpecialWindow   bool
	IsActive             *bool
	ImageFile            *multipart.FileHeader
}

type ListProductsQuery struct {
	Page        int
	Limit       int
	StockStatus string
	SortBy      string // "created_at" | "updated_at" — default: created_at
}

type ProductListResult struct {
	Items      []Product `json:"items"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	Total      int64     `json:"total"`
	TotalPages int       `json:"total_pages"`
	HasNext    bool      `json:"has_next"`
	HasPrev    bool      `json:"has_prev"`
}

type GenerateMissingSKUResult struct {
	UpdatedCount int `json:"updated_count"`
}
