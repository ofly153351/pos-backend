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

// WarehouseProduct represents a product with stock in a warehouse.
// Read-only DTO populated from stocks JOIN locations JOIN product_view.
type WarehouseProduct struct {
	ProductID       string  `json:"product_id" gorm:"column:product_id"`
	ProductName     string  `json:"product_name" gorm:"column:product_name"`
	ProductSKU      string  `json:"product_sku" gorm:"column:product_sku"`
	ProductBarcode  string  `json:"product_barcode" gorm:"column:product_barcode"`
	ProductPrice    float64 `json:"product_price" gorm:"column:product_price"`
	CostPrice       float64 `json:"cost_price" gorm:"column:cost_price"`
	EffectivePrice  float64 `json:"effective_price" gorm:"column:effective_price"`
	ImageURL        string  `json:"image_url" gorm:"column:image_url"`
	ProductTypeName string  `json:"product_type_name" gorm:"column:product_type_name"`
	ProductUnitName string  `json:"product_unit_name" gorm:"column:product_unit_name"`
	ProductMinStock int     `json:"product_min_stock" gorm:"column:product_min_stock"`
	ProductMaxStock *int    `json:"product_max_stock,omitempty" gorm:"column:product_max_stock"`
	Quantity        int     `json:"quantity" gorm:"column:quantity"`
}

// TableName returns empty — this is a read-only DTO, not a real table.
func (WarehouseProduct) TableName() string {
	return ""
}

// AddWarehouseProductRequest is the request to add a product to a warehouse.
type AddWarehouseProductRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type UpdateWarehouseProductRequest struct {
	Quantity *int `json:"quantity"`
}

// WarehouseTransferRequest is the request to transfer stock from a warehouse
// to another warehouse or to a sale_point location.
type WarehouseTransferRequest struct {
	ProductID       string `json:"product_id"`
	Quantity        int    `json:"quantity"`
	DestinationType string `json:"destination_type"` // "warehouse" or "stock"
	DestinationID   string `json:"destination_id"`   // warehouse_id if destination_type="warehouse", ignored if "stock"
	Note            string `json:"note"`
}
