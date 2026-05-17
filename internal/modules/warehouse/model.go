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
// Two modes:
//   1. Reference mode — ProductID is not null, links to an existing product in the products table
//   2. Standalone mode — ProductID is null, product info stored directly in name/sku/barcode/price/unit_name/type_name
type WarehouseProduct struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	WarehouseID string    `json:"warehouse_id" gorm:"column:warehouse_id"`
	ProductID   *string   `json:"product_id" gorm:"column:product_id"`
	Quantity    int       `json:"quantity" gorm:"column:quantity"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`

	// Standalone product columns (written directly, used when ProductID is null)
	StandaloneName     string  `json:"standalone_name,omitempty" gorm:"column:name"`
	StandaloneSKU      string  `json:"standalone_sku,omitempty" gorm:"column:sku"`
	StandaloneBarcode  string  `json:"standalone_barcode,omitempty" gorm:"column:barcode"`
	StandalonePrice    float64 `json:"standalone_price,omitempty" gorm:"column:price"`
	StandaloneUnitName string  `json:"standalone_unit_name,omitempty" gorm:"column:unit_name"`
	StandaloneTypeName string  `json:"standalone_type_name,omitempty" gorm:"column:type_name"`

	// Joined fields from product_view (read-only, populated via LEFT JOIN)
	ProductName      string  `json:"product_name,omitempty" gorm:"column:product_name;->"`
	ProductSKU       string  `json:"product_sku,omitempty" gorm:"column:product_sku;->"`
	ProductBarcode   string  `json:"product_barcode,omitempty" gorm:"column:product_barcode;->"`
	ProductPrice     float64 `json:"product_price,omitempty" gorm:"column:product_price;->"`
	ImageURL         string  `json:"image_url,omitempty" gorm:"column:image_url;->"`
	ProductTypeName  string  `json:"product_type_name,omitempty" gorm:"column:product_type_name;->"`
	ProductUnitName  string  `json:"product_unit_name,omitempty" gorm:"column:product_unit_name;->"`
	ProductMinStock  int     `json:"product_min_stock,omitempty" gorm:"column:product_min_stock;->"`
	ProductMaxStock  *int    `json:"product_max_stock,omitempty" gorm:"column:product_max_stock;->"`
	ProductQuantity  int     `json:"product_quantity,omitempty" gorm:"column:product_quantity;->"`
}

// IsStandalone returns true when this warehouse product has no reference to the products table.
func (wp WarehouseProduct) IsStandalone() bool {
	return wp.ProductID == nil || *wp.ProductID == ""
}

// DisplayName returns the effective name (from joined product_view or standalone field).
func (wp WarehouseProduct) DisplayName() string {
	if wp.IsStandalone() {
		return wp.StandaloneName
	}
	return wp.ProductName
}

// DisplaySKU returns the effective SKU.
func (wp WarehouseProduct) DisplaySKU() string {
	if wp.IsStandalone() {
		return wp.StandaloneSKU
	}
	return wp.ProductSKU
}

func (WarehouseProduct) TableName() string {
	return "warehouse_products"
}

// AddWarehouseProductRequest supports both modes:
//   - product_id set → reference mode (links to existing product)
//   - product_id null/empty → standalone mode (name is required)
type AddWarehouseProductRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`

	// Standalone fields (required when product_id is empty)
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Barcode  string  `json:"barcode"`
	Price    float64 `json:"price"`
	UnitName string  `json:"unit_name"`
	TypeName string  `json:"type_name"`
}

// IsStandalone returns true when adding a standalone product (no product_id ref).
func (r AddWarehouseProductRequest) IsStandalone() bool {
	return r.ProductID == ""
}

type UpdateWarehouseProductRequest struct {
	Quantity *int `json:"quantity"`
}
