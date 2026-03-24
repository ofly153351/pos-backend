package product

import (
	"mime/multipart"
	"time"
)

const (
	UnitTypePiece = "piece"
)

type Product struct {
	ID                  string     `json:"id" gorm:"column:id;primaryKey"`
	StoreID             string     `json:"store_id" gorm:"column:store_id"`
	ProductTypeID       string     `json:"product_type_id,omitempty" gorm:"column:product_type_id"`
	ProductTypeName     string     `json:"product_type_name,omitempty" gorm:"-"`
	Name                string     `json:"name" gorm:"column:name"`
	SKU                 string     `json:"sku,omitempty" gorm:"column:sku"`
	UnitType            string     `json:"unit_type" gorm:"column:unit_type"`
	ImageURL            string     `json:"image_url,omitempty" gorm:"column:image_url"`
	Quantity            int        `json:"quantity" gorm:"column:quantity"`
	BasePrice           float64    `json:"base_price" gorm:"column:base_price"`
	SpecialPrice        *float64   `json:"special_price,omitempty" gorm:"column:special_price"`
	SpecialPriceStartAt *time.Time `json:"special_price_start_at,omitempty" gorm:"column:special_price_start_at"`
	SpecialPriceEndAt   *time.Time `json:"special_price_end_at,omitempty" gorm:"column:special_price_end_at"`
	EffectivePrice      float64    `json:"effective_price" gorm:"-"`
	IsActive            bool       `json:"is_active" gorm:"column:is_active"`
	CreatedAt           time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

func (Product) TableName() string {
	return "products"
}

type CreateProductRequest struct {
	Name                string
	SKU                 string
	ProductTypeID       string
	UnitType            string
	Quantity            *int
	BasePrice           float64
	SpecialPrice        *float64
	SpecialPriceStartAt *time.Time
	SpecialPriceEndAt   *time.Time
	IsActive            *bool
	ImageFile           *multipart.FileHeader
}

type UpdateProductRequest struct {
	Name                *string
	SKU                 *string
	ProductTypeID       *string
	UnitType            *string
	Quantity            *int
	BasePrice           *float64
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
