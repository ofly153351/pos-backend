package product

import (
	"mime/multipart"
	"time"
)

type Product struct {
	ID                  string     `json:"id" gorm:"column:id;primaryKey"`
	StoreID             string     `json:"store_id" gorm:"column:store_id"`
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
	IsActive            bool       `json:"is_active" gorm:"column:is_active"`
	CreatedAt           time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"column:updated_at"`
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
	ProductTypeID       string
	ProductUnitID       string
	MinStock            *int
	MaxStock            *int
	BasePrice           float64
	CostPrice           float64
	SpecialPrice        *float64
	SpecialPriceStartAt *time.Time
	SpecialPriceEndAt   *time.Time
	IsActive            *bool
	ImageFile           *multipart.FileHeader
}

type UpdateProductRequest struct {
	Name                *string
	BrandID             *string
	SKU                 *string
	Barcode             *string
	ProductCode         *string
	Description         *string
	StorageLocation     *string
	ClearBarcode        bool
	ClearSKU            bool
	ClearProductCode    bool
	ProductTypeID       *string
	ProductUnitID       *string
	MinStock            *int
	MaxStock            *int
	ClearMaxStock       bool
	BasePrice           *float64
	CostPrice           *float64
	SpecialPrice        *float64
	ClearSpecialPrice   bool
	SpecialPriceStartAt *time.Time
	SpecialPriceEndAt   *time.Time
	ClearSpecialWindow  bool
	IsActive            *bool
	ImageFile           *multipart.FileHeader
}

type ListProductsQuery struct {
	Page  int
	Limit int
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
