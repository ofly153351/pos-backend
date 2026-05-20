package location

import "time"

type Location struct {
	ID           string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID      string    `json:"store_id" gorm:"column:store_id"`
	WarehouseID  string    `json:"warehouse_id" gorm:"column:warehouse_id"`
	Name         string    `json:"name" gorm:"column:name"`
	Code         string    `json:"code,omitempty" gorm:"column:code"`
	IsSalePoint  bool      `json:"is_sale_point" gorm:"column:is_sale_point"`
	IsActive     bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Relations
	WarehouseName string `json:"warehouse_name,omitempty" gorm:"-"`
}

func (Location) TableName() string { return "locations" }

type CreateLocationRequest struct {
	WarehouseID string `json:"warehouse_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	IsSalePoint bool   `json:"is_sale_point"`
}

type UpdateLocationRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	IsSalePoint *bool   `json:"is_sale_point"`
	IsActive    *bool   `json:"is_active"`
}
