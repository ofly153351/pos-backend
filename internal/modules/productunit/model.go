package productunit

import "time"

type ProductUnit struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID     string    `json:"store_id" gorm:"column:store_id"`
	Name        string    `json:"name" gorm:"column:name"`
	Description string    `json:"description,omitempty" gorm:"column:description"`
	IsActive    bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (ProductUnit) TableName() string {
	return "product_units"
}

type CreateProductUnitRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateProductUnitRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}
