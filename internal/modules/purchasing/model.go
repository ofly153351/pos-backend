package purchasing

import "time"

type Supplier struct {
	ID            string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID       string    `json:"store_id" gorm:"column:store_id"`
	Name          string    `json:"name" gorm:"column:name"`
	Phone         string    `json:"phone,omitempty" gorm:"column:phone"`
	Address       string    `json:"address,omitempty" gorm:"column:address"`
	TaxID         string    `json:"tax_id,omitempty" gorm:"column:tax_id"`
	ContactPerson string    `json:"contact_person,omitempty" gorm:"column:contact_person"`
	Note          string    `json:"note,omitempty" gorm:"column:note"`
	IsActive      bool      `json:"is_active" gorm:"column:is_active"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (Supplier) TableName() string {
	return "suppliers"
}

type PurchaseOrderStatus string

const (
	POStatusPending   PurchaseOrderStatus = "pending"
	POStatusPartial   PurchaseOrderStatus = "partial"
	POStatusCompleted PurchaseOrderStatus = "completed"
	POStatusCancelled PurchaseOrderStatus = "cancelled"
)

type PurchaseOrder struct {
	ID         string              `json:"id" gorm:"column:id;primaryKey"`
	StoreID    string              `json:"store_id" gorm:"column:store_id"`
	SupplierID string              `json:"supplier_id,omitempty" gorm:"column:supplier_id"`
	OrderNumber string             `json:"order_number" gorm:"column:order_number"`
	Status     PurchaseOrderStatus `json:"status" gorm:"column:status"`
	Notes      string              `json:"notes,omitempty" gorm:"column:notes"`
	TotalCost  float64             `json:"total_cost" gorm:"column:total_cost"`
	ReceivedAt *time.Time          `json:"received_at,omitempty" gorm:"column:received_at"`
	CreatedAt  time.Time           `json:"created_at" gorm:"column:created_at"`
	UpdatedAt  time.Time           `json:"updated_at" gorm:"column:updated_at"`

	// Relations
	Supplier *Supplier          `json:"supplier,omitempty" gorm:"foreignKey:SupplierID;references:ID"`
	Items    []PurchaseOrderItem `json:"items,omitempty" gorm:"foreignKey:PurchaseOrderID;references:ID"`
}

func (PurchaseOrder) TableName() string {
	return "purchase_orders"
}

type PurchaseOrderItem struct {
	ID               string    `json:"id" gorm:"column:id;primaryKey"`
	PurchaseOrderID  string    `json:"purchase_order_id" gorm:"column:purchase_order_id"`
	ProductID        string    `json:"product_id" gorm:"column:product_id"`
	Quantity         int       `json:"quantity" gorm:"column:quantity"`
	ReceivedQuantity int       `json:"received_quantity" gorm:"column:received_quantity"`
	UnitCost         float64   `json:"unit_cost" gorm:"column:unit_cost"`
	LineTotal        float64   `json:"line_total" gorm:"column:line_total"`
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at"`

	// Relations
	ProductName string `json:"product_name,omitempty" gorm:"-"`
}

func (PurchaseOrderItem) TableName() string {
	return "purchase_order_items"
}

type CreateSupplierRequest struct {
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	TaxID         string `json:"tax_id"`
	ContactPerson string `json:"contact_person"`
	Note          string `json:"note"`
	IsActive      *bool  `json:"is_active"`
}

type UpdateSupplierRequest struct {
	Name          *string `json:"name"`
	Phone         *string `json:"phone"`
	Address       *string `json:"address"`
	TaxID         *string `json:"tax_id"`
	ContactPerson *string `json:"contact_person"`
	Note          *string `json:"note"`
	IsActive      *bool   `json:"is_active"`
}

type CreatePOItemRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitCost  float64 `json:"unit_cost"`
}

type CreatePORequest struct {
	SupplierID string              `json:"supplier_id"`
	Notes      string              `json:"notes"`
	Items      []CreatePOItemRequest `json:"items"`
}

type UpdatePORequest struct {
	SupplierID *string `json:"supplier_id"`
	Notes      *string `json:"notes"`
	Status     *string `json:"status"`
}

type ReceivePOItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ReceivePORequest struct {
	Items []ReceivePOItemRequest `json:"items"`
}

type SupplierProduct struct {
	ID            string    `json:"id" gorm:"column:id;primaryKey"`
	SupplierID    string    `json:"supplier_id" gorm:"column:supplier_id"`
	ProductID     string    `json:"product_id" gorm:"column:product_id"`
	SupplierSKU   string    `json:"supplier_sku" gorm:"column:supplier_sku"`
	SupplierPrice float64   `json:"supplier_price" gorm:"column:supplier_price"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (SupplierProduct) TableName() string {
	return "supplier_products"
}

type SupplierProductResponse struct {
	ID            string  `json:"id"`
	SupplierID    string  `json:"supplier_id"`
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	ProductSKU    string  `json:"product_sku"`
	SupplierSKU   string  `json:"supplier_sku"`
	SupplierPrice float64 `json:"supplier_price"`
	CreatedAt     string  `json:"created_at"`
}

type AddSupplierProductRequest struct {
	ProductID     string  `json:"product_id"`
	SupplierSKU   string  `json:"supplier_sku"`
	SupplierPrice float64 `json:"supplier_price"`
}

type UpdateSupplierProductRequest struct {
	SupplierSKU   string  `json:"supplier_sku"`
	SupplierPrice float64 `json:"supplier_price"`
}

type CreateSupplierProductAndLinkRequest struct {
	Name          string  `json:"name"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode"`
	ProductTypeID string  `json:"product_type_id"`
	ProductUnitID string  `json:"product_unit_id"`
	BasePrice     float64 `json:"base_price"`
	SupplierSKU   string  `json:"supplier_sku"`
	SupplierPrice float64 `json:"supplier_price"`
}

type PurchaseOrderWithSupplier struct {
	PurchaseOrder
	SupplierName string `json:"supplier_name,omitempty"`
}
