package product

import (
	"mime/multipart"
	"time"
)

const (
	UnitTypePiece = "piece"
)

type Product struct {
	ID                  string     `json:"id"`
	StoreID             string     `json:"store_id"`
	ProductTypeID       string     `json:"product_type_id,omitempty"`
	ProductTypeName     string     `json:"product_type_name,omitempty"`
	Name                string     `json:"name"`
	SKU                 string     `json:"sku,omitempty"`
	UnitType            string     `json:"unit_type"`
	ImageURL            string     `json:"image_url,omitempty"`
	Quantity            int        `json:"quantity"`
	BasePrice           float64    `json:"base_price"`
	SpecialPrice        *float64   `json:"special_price,omitempty"`
	SpecialPriceStartAt *time.Time `json:"special_price_start_at,omitempty"`
	SpecialPriceEndAt   *time.Time `json:"special_price_end_at,omitempty"`
	EffectivePrice      float64    `json:"effective_price"`
	IsActive            bool       `json:"is_active"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
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
