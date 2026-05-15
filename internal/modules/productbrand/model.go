package productbrand

import "time"

type ProductBrand struct {
	ID        string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID   string    `json:"store_id" gorm:"column:store_id"`
	Name      string    `json:"name" gorm:"column:name"`
	IsActive  bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (ProductBrand) TableName() string {
	return "product_brands"
}

type CreateProductBrandRequest struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
}

type UpdateProductBrandRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}
