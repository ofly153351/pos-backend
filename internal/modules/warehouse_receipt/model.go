package warehouse_receipt

import (
	"mime/multipart"
	"time"
)

type ReceiptStatus string

const (
	ReceiptStatusDraft     ReceiptStatus = "draft"
	ReceiptStatusConfirmed ReceiptStatus = "confirmed"
	ReceiptStatusCancelled ReceiptStatus = "cancelled"
)

type WarehouseReceipt struct {
	ID                 string        `json:"id" gorm:"column:id;primaryKey"`
	StoreID            string        `json:"store_id" gorm:"column:store_id"`
	WarehouseID        string        `json:"warehouse_id" gorm:"column:warehouse_id"`
	SupplierID         string        `json:"supplier_id,omitempty" gorm:"column:supplier_id"`
	PurchaseOrderID    string        `json:"purchase_order_id,omitempty" gorm:"column:purchase_order_id"`
	DocumentNo         string        `json:"document_no" gorm:"column:document_no"`
	Status             ReceiptStatus `json:"status" gorm:"column:status"`
	ReceivedAt         time.Time     `json:"received_at" gorm:"column:received_at"`
	ReferenceNo        string        `json:"reference_no,omitempty" gorm:"column:reference_no"`
	Note               string        `json:"note,omitempty" gorm:"column:note"`
	VATIncluded        bool          `json:"vat_included" gorm:"column:vat_included"`
	VATPercent         float64       `json:"vat_percent" gorm:"column:vat_percent"`
	TotalItems         int           `json:"total_items" gorm:"column:total_items"`
	SubtotalAmount     float64       `json:"subtotal_amount" gorm:"column:subtotal_amount"`
	DiscountAmount     float64       `json:"discount_amount" gorm:"column:discount_amount"`
	NetAmount          float64       `json:"net_amount" gorm:"column:net_amount"`
	VATAmount          float64       `json:"vat_amount" gorm:"column:vat_amount"`
	TotalAmount        float64       `json:"total_amount" gorm:"column:total_amount"`
	AttachmentURL      string        `json:"attachment_url,omitempty" gorm:"column:attachment_url"`
	AttachmentMimeType string        `json:"attachment_mime_type,omitempty" gorm:"column:attachment_mime_type"`
	AttachmentName     string        `json:"attachment_name,omitempty" gorm:"column:attachment_name"`
	AttachmentSize     int64         `json:"attachment_size,omitempty" gorm:"column:attachment_size"`
	CreatedBy          string        `json:"created_by" gorm:"column:created_by"`
	ConfirmedBy        string        `json:"confirmed_by,omitempty" gorm:"column:confirmed_by"`
	CancelledBy        string        `json:"cancelled_by,omitempty" gorm:"column:cancelled_by"`
	ConfirmedAt        *time.Time    `json:"confirmed_at,omitempty" gorm:"column:confirmed_at"`
	CancelledAt        *time.Time    `json:"cancelled_at,omitempty" gorm:"column:cancelled_at"`
	CreatedAt          time.Time     `json:"created_at" gorm:"column:created_at"`
	UpdatedAt          time.Time     `json:"updated_at" gorm:"column:updated_at"`

	WarehouseName      string                              `json:"warehouse_name,omitempty" gorm:"-"`
	SupplierName       string                              `json:"supplier_name,omitempty" gorm:"-"`
	CreatedByName      string                              `json:"created_by_name,omitempty" gorm:"-"`
	ConfirmedByName    string                              `json:"confirmed_by_name,omitempty" gorm:"-"`
	CancelledByName    string                              `json:"cancelled_by_name,omitempty" gorm:"-"`
	PurchaseOrderNo    string                              `json:"purchase_order_no,omitempty" gorm:"-"`
	Items              []WarehouseReceiptItem              `json:"items" gorm:"-"`
	Preview            []StockImpactPreview                `json:"stock_preview" gorm:"-"`
	Audits             []WarehouseReceiptAudit             `json:"audits" gorm:"-"`
	Attachments        []WarehouseReceiptAttachment        `json:"attachments" gorm:"-"`
	PendingAttachments []WarehouseReceiptPendingAttachment `json:"pending_attachments" gorm:"-"`
}

func (WarehouseReceipt) TableName() string { return "warehouse_receipts" }

type WarehouseReceiptItem struct {
	ID             string    `json:"id" gorm:"column:id;primaryKey"`
	ReceiptID      string    `json:"receipt_id" gorm:"column:receipt_id"`
	ProductID      string    `json:"product_id" gorm:"column:product_id"`
	LocationID     string    `json:"location_id" gorm:"column:location_id"`
	WarehouseID    string    `json:"warehouse_id" gorm:"column:warehouse_id"`
	ZoneName       string    `json:"zone_name,omitempty" gorm:"column:zone_name"`
	FloorName      string    `json:"floor_name,omitempty" gorm:"column:floor_name"`
	LocationName   string    `json:"location_name,omitempty" gorm:"column:location_name"`
	ProductName    string    `json:"product_name" gorm:"column:product_name"`
	SKU            string    `json:"sku,omitempty" gorm:"column:sku"`
	Barcode        string    `json:"barcode,omitempty" gorm:"column:barcode"`
	UnitName       string    `json:"unit_name,omitempty" gorm:"column:unit_name"`
	Quantity       int       `json:"quantity" gorm:"column:quantity"`
	UnitPrice      float64   `json:"unit_price" gorm:"column:unit_price"`
	DiscountType   string    `json:"discount_type,omitempty" gorm:"column:discount_type"`
	DiscountValue  *float64  `json:"discount_value,omitempty" gorm:"column:discount_value"`
	DiscountAmount float64   `json:"discount_amount" gorm:"column:discount_amount"`
	LineSubtotal   float64   `json:"line_subtotal" gorm:"column:line_subtotal"`
	LineNet        float64   `json:"line_net" gorm:"column:line_net"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (WarehouseReceiptItem) TableName() string { return "warehouse_receipt_items" }

type WarehouseReceiptAudit struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	ReceiptID   string    `json:"receipt_id" gorm:"column:receipt_id"`
	Action      string    `json:"action" gorm:"column:action"`
	Description string    `json:"description,omitempty" gorm:"column:description"`
	ActorID     string    `json:"actor_id" gorm:"column:actor_id"`
	ActorName   string    `json:"actor_name,omitempty" gorm:"-"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
}

func (WarehouseReceiptAudit) TableName() string { return "warehouse_receipt_audits" }

type CreateReceiptRequest struct {
	StoreID         string     `json:"store_id"`
	WarehouseID     string     `json:"warehouse_id"`
	SupplierID      string     `json:"supplier_id"`
	PurchaseOrderID string     `json:"purchase_order_id"`
	DocumentNo      string     `json:"document_no"`
	ReceivedAt      *time.Time `json:"received_at"`
	ReferenceNo     string     `json:"reference_no"`
	Note            string     `json:"note"`
	VATIncluded     *bool      `json:"vat_included"`
	VATPercent      *float64   `json:"vat_percent"`
}

type UpdateReceiptRequest struct {
	WarehouseID     *string    `json:"warehouse_id"`
	SupplierID      *string    `json:"supplier_id"`
	PurchaseOrderID *string    `json:"purchase_order_id"`
	DocumentNo      *string    `json:"document_no"`
	ReceivedAt      *time.Time `json:"received_at"`
	ReferenceNo     *string    `json:"reference_no"`
	Note            *string    `json:"note"`
	VATIncluded     *bool      `json:"vat_included"`
	VATPercent      *float64   `json:"vat_percent"`
}

type ListReceiptsQuery struct {
	StoreID string
	Status  *ReceiptStatus
	Page    int
	Limit   int
}

type ReceiptListResult struct {
	Items      []WarehouseReceipt `json:"items"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	Total      int64              `json:"total"`
	TotalPages int                `json:"total_pages"`
	HasNext    bool               `json:"has_next"`
	HasPrev    bool               `json:"has_prev"`
}

type ReceiptItemInput struct {
	ProductID     string   `json:"product_id"`
	LocationID    string   `json:"location_id"`
	Quantity      int      `json:"quantity"`
	UnitPrice     *float64 `json:"unit_price"`
	DiscountType  string   `json:"discount_type"`
	DiscountValue *float64 `json:"discount_value"`
}

type AddReceiptItemsRequest struct {
	ReplaceExisting bool               `json:"replace_existing"`
	Items           []ReceiptItemInput `json:"items"`
}

type UpdateReceiptItemRequest struct {
	ProductID     *string  `json:"product_id"`
	LocationID    *string  `json:"location_id"`
	Quantity      *int     `json:"quantity"`
	UnitPrice     *float64 `json:"unit_price"`
	DiscountType  *string  `json:"discount_type"`
	DiscountValue *float64 `json:"discount_value"`
}

type GenerateDocumentNoRequest struct {
	StoreID string `json:"store_id"`
}

type GenerateDocumentNoResponse struct {
	DocumentNo string `json:"document_no"`
}

type UploadAttachmentRequest struct {
	File *multipart.FileHeader
}

type WarehouseReceiptAttachment struct {
	ID         string    `json:"id" gorm:"column:id;primaryKey"`
	ReceiptID  string    `json:"receipt_id" gorm:"column:receipt_id"`
	URL        string    `json:"url" gorm:"column:url"`
	MimeType   string    `json:"mime_type" gorm:"column:mime_type"`
	Name       string    `json:"name" gorm:"column:name"`
	Size       int64     `json:"size" gorm:"column:size"`
	UploadedBy string    `json:"uploaded_by" gorm:"column:uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at" gorm:"column:uploaded_at"`
}

type WarehouseReceiptPendingAttachment struct {
	ID        string    `json:"id" gorm:"column:id;primaryKey"`
	ReceiptID string    `json:"receipt_id" gorm:"column:receipt_id"`
	MimeType  string    `json:"mime_type" gorm:"column:mime_type"`
	Name      string    `json:"name" gorm:"column:name"`
	Size      int64     `json:"size" gorm:"column:size"`
	CreatedBy string    `json:"created_by" gorm:"column:created_by"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	Data      []byte    `json:"-" gorm:"column:data"`
}

type StockImpactPreview struct {
	ItemID         string  `json:"item_id"`
	ProductID      string  `json:"product_id"`
	ProductName    string  `json:"product_name"`
	LocationID     string  `json:"location_id"`
	LocationName   string  `json:"location_name"`
	Quantity       int     `json:"quantity"`
	BeforeQuantity int     `json:"before_quantity"`
	AfterQuantity  int     `json:"after_quantity"`
	BeforeValue    float64 `json:"before_value"`
	AfterValue     float64 `json:"after_value"`
	ValueChange    float64 `json:"value_change"`
}

type PrintReceiptResponse struct {
	ReceiptID   string `json:"receipt_id"`
	DocumentNo  string `json:"document_no"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	HTML        string `json:"html"`
}

type receiptProductSnapshot struct {
	ID        string  `gorm:"column:id"`
	StoreID   string  `gorm:"column:store_id"`
	Name      string  `gorm:"column:name"`
	SKU       string  `gorm:"column:sku"`
	Barcode   string  `gorm:"column:barcode"`
	UnitName  string  `gorm:"column:unit_name"`
	IsActive  bool    `gorm:"column:is_active"`
	CostPrice float64 `gorm:"column:cost_price"`
}

type locationSnapshot struct {
	ID          string `gorm:"column:id"`
	StoreID     string `gorm:"column:store_id"`
	WarehouseID string `gorm:"column:warehouse_id"`
	Name        string `gorm:"column:name"`
	ZoneName    string `gorm:"column:zone_name"`
	FloorName   string `gorm:"column:floor_name"`
	IsActive    bool   `gorm:"column:is_active"`
	IsSalePoint bool   `gorm:"column:is_sale_point"`
}

type purchaseOrderSnapshot struct {
	ID          string `gorm:"column:id"`
	StoreID     string `gorm:"column:store_id"`
	SupplierID  string `gorm:"column:supplier_id"`
	OrderNumber string `gorm:"column:order_number"`
	Status      string `gorm:"column:status"`
}

type purchaseOrderItemSnapshot struct {
	ID               string  `gorm:"column:id"`
	PurchaseOrderID  string  `gorm:"column:purchase_order_id"`
	ProductID        string  `gorm:"column:product_id"`
	Quantity         int     `gorm:"column:quantity"`
	ReceivedQuantity int     `gorm:"column:received_quantity"`
	UnitCost         float64 `gorm:"column:unit_cost"`
}
