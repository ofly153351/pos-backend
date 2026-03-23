package invoice

import "time"

const (
	StatusUnpaid        = "unpaid"
	StatusPartiallyPaid = "partially_paid"
	StatusPaid          = "paid"
	StatusCancelled     = "cancelled"
)

type Invoice struct {
	ID                     string           `json:"id" gorm:"column:id;primaryKey"`
	StoreID                string           `json:"store_id" gorm:"column:store_id"`
	StoreName              string           `json:"store_name,omitempty" gorm:"column:store_name"`
	InvoiceNumber          string           `json:"invoice_number" gorm:"column:invoice_number"`
	CustomerID             string           `json:"customer_id" gorm:"column:customer_id"`
	CustomerName           string           `json:"customer_name,omitempty" gorm:"column:customer_name"`
	CashierUserID          string           `json:"cashier_user_id" gorm:"column:cashier_user_id"`
	Status                 string           `json:"status" gorm:"column:status"`
	PaymentMethod          string           `json:"payment_method,omitempty" gorm:"column:payment_method"`
	Note                   string           `json:"note,omitempty" gorm:"column:note"`
	DueAt                  *time.Time       `json:"due_at,omitempty" gorm:"column:due_at"`
	CustomerLevel          *int             `json:"customer_level,omitempty" gorm:"column:customer_level"`
	NetworkDiscountPercent float64          `json:"network_discount_percent" gorm:"column:network_discount_percent"`
	TotalItems             int              `json:"total_items" gorm:"column:total_items"`
	SubtotalAmount         float64          `json:"subtotal_amount" gorm:"column:subtotal_amount"`
	DiscountAmount         float64          `json:"discount_amount" gorm:"column:discount_amount"`
	TotalAmount            float64          `json:"total_amount" gorm:"column:total_amount"`
	PaidAmount             float64          `json:"paid_amount" gorm:"column:paid_amount"`
	RemainingAmount        float64          `json:"remaining_amount" gorm:"column:remaining_amount"`
	CreatedAt              time.Time        `json:"created_at" gorm:"column:created_at"`
	UpdatedAt              time.Time        `json:"updated_at" gorm:"column:updated_at"`
	Items                  []InvoiceItem    `json:"items,omitempty" gorm:"foreignKey:InvoiceID;references:ID"`
	Payments               []InvoicePayment `json:"payments,omitempty" gorm:"foreignKey:InvoiceID;references:ID"`
}

type InvoiceItem struct {
	ID                    string    `json:"id" gorm:"column:id;primaryKey"`
	InvoiceID             string    `json:"invoice_id" gorm:"column:invoice_id"`
	ProductID             string    `json:"product_id" gorm:"column:product_id"`
	ProductName           string    `json:"product_name" gorm:"column:product_name"`
	SKU                   string    `json:"sku,omitempty" gorm:"column:sku"`
	UnitType              string    `json:"unit_type,omitempty" gorm:"column:unit_type"`
	Quantity              int       `json:"quantity" gorm:"column:quantity"`
	UnitPrice             float64   `json:"unit_price" gorm:"column:unit_price"`
	DiscountType          string    `json:"discount_type,omitempty" gorm:"column:discount_type"`
	DiscountValue         *float64  `json:"discount_value,omitempty" gorm:"column:discount_value"`
	DiscountAmountPerUnit float64   `json:"discount_amount_per_unit" gorm:"column:discount_amount_per_unit"`
	LineSubtotal          float64   `json:"line_subtotal" gorm:"column:line_subtotal"`
	LineDiscountTotal     float64   `json:"line_discount_total" gorm:"column:line_discount_total"`
	LineTotal             float64   `json:"line_total" gorm:"column:line_total"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
}

type InvoicePayment struct {
	ID            string    `json:"id" gorm:"column:id;primaryKey"`
	InvoiceID     string    `json:"invoice_id" gorm:"column:invoice_id"`
	PaidAmount    float64   `json:"paid_amount" gorm:"column:paid_amount"`
	PaymentMethod string    `json:"payment_method" gorm:"column:payment_method"`
	Note          string    `json:"note,omitempty" gorm:"column:note"`
	PaidAt        time.Time `json:"paid_at" gorm:"column:paid_at"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
}

func (Invoice) TableName() string        { return "invoices" }
func (InvoiceItem) TableName() string    { return "invoice_items" }
func (InvoicePayment) TableName() string { return "invoice_payments" }

type CreateInvoiceRequest struct {
	CustomerID string                     `json:"customer_id"`
	DueAt      *time.Time                 `json:"due_at"`
	Note       string                     `json:"note"`
	Items      []CreateInvoiceItemRequest `json:"items"`
}

type CreateInvoiceItemRequest struct {
	ProductID     string   `json:"product_id"`
	Quantity      int      `json:"quantity"`
	DiscountType  string   `json:"discount_type"`
	DiscountValue *float64 `json:"discount_value"`
}

type CreateInvoicePaymentRequest struct {
	PaidAmount    float64 `json:"paid_amount"`
	PaymentMethod string  `json:"payment_method"`
	Note          string  `json:"note"`
}

type productSnapshot struct {
	ID                  string
	Name                string
	SKU                 string
	UnitType            string
	Quantity            int
	IsActive            bool
	BasePrice           float64
	SpecialPrice        *float64
	SpecialPriceStartAt *time.Time
	SpecialPriceEndAt   *time.Time
}
