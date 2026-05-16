package warehouse

import "time"

type Warehouse struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID     string    `json:"store_id" gorm:"column:store_id"`
	Name        string    `json:"name" gorm:"column:name"`
	Code        string    `json:"code,omitempty" gorm:"column:code"`
	Address     string    `json:"address,omitempty" gorm:"column:address"`
	Phone       string    `json:"phone,omitempty" gorm:"column:phone"`
	ContactName string    `json:"contact_name,omitempty" gorm:"column:contact_name"`
	IsActive    bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (Warehouse) TableName() string {
	return "warehouses"
}

type CreateWarehouseRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	ContactName string `json:"contact_name"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateWarehouseRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone"`
	ContactName *string `json:"contact_name"`
	IsActive    *bool   `json:"is_active"`
}

// WarehouseProduct represents a product assigned to a warehouse.
type WarehouseProduct struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	WarehouseID string    `json:"warehouse_id" gorm:"column:warehouse_id"`
	ProductID   string    `json:"product_id" gorm:"column:product_id"`
	Quantity    int       `json:"quantity" gorm:"column:quantity"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`

	// Joined fields from products table
	ProductName  string  `json:"product_name,omitempty" gorm:"column:product_name;->"`
	ProductSKU   string  `json:"product_sku,omitempty" gorm:"column:product_sku;->"`
	ProductPrice float64 `json:"product_price,omitempty" gorm:"column:product_price;->"`
	ImageURL     string  `json:"image_url,omitempty" gorm:"column:image_url;->"`
}

func (WarehouseProduct) TableName() string {
	return "warehouse_products"
}

type AddWarehouseProductRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}
