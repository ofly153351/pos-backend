package sale

import "time"

type Sale struct {
	ID                     string     `json:"id" gorm:"column:id;primaryKey"`
	StoreID                string     `json:"store_id" gorm:"column:store_id"`
	SaleNumber             string     `json:"sale_number" gorm:"column:sale_number"`
	CashierUserID          string     `json:"cashier_user_id" gorm:"column:cashier_user_id"`
	CashierName            string     `json:"cashier_name,omitempty" gorm:"column:cashier_name"`
	Status                 string     `json:"status" gorm:"column:status"`
	PaymentMethod          string     `json:"payment_method" gorm:"column:payment_method"`
	Note                   string     `json:"note,omitempty" gorm:"column:note"`
	CustomerID             string     `json:"customer_id,omitempty" gorm:"column:customer_id"`
	CustomerName           string     `json:"customer_name,omitempty" gorm:"column:customer_name"`
	CustomerPhone          string     `json:"customer_phone,omitempty" gorm:"column:customer_phone"`
	StoreName              string     `json:"store_name,omitempty" gorm:"column:store_name"`
	StoreAddress           string     `json:"store_address,omitempty" gorm:"column:store_address"`
	StorePhone             string     `json:"store_phone,omitempty" gorm:"column:store_phone"`
	CustomerLevel          *int       `json:"customer_level,omitempty" gorm:"column:customer_level"`
	NetworkDiscountPercent float64    `json:"network_discount_percent" gorm:"column:network_discount_percent"`
	TotalItems             int        `json:"total_items" gorm:"column:total_items"`
	SubtotalAmount         float64    `json:"subtotal_amount" gorm:"column:subtotal_amount"`
	DiscountAmount         float64    `json:"discount_amount" gorm:"column:discount_amount"`
	BillDiscountAmount     float64    `json:"bill_discount_amount" gorm:"column:bill_discount_amount"`
	VATIncluded            bool       `json:"vat_included" gorm:"column:vat_included"`
	VATPercent             float64    `json:"vat_percent" gorm:"column:vat_percent"`
	VATAmount              float64    `json:"vat_amount" gorm:"column:vat_amount"`
	TotalAmount            float64    `json:"total_amount" gorm:"column:total_amount"`
	PaidAmount             float64    `json:"paid_amount" gorm:"column:paid_amount"`
	ChangeAmount           float64    `json:"change_amount" gorm:"column:change_amount"`
	SoldAt                 time.Time  `json:"sold_at" gorm:"column:sold_at"`
	CreatedAt              time.Time  `json:"created_at" gorm:"column:created_at"`
	Items                  []SaleItem `json:"items,omitempty" gorm:"foreignKey:SaleID;references:ID"`
}

type SaleItem struct {
	ID                    string    `json:"id" gorm:"column:id;primaryKey"`
	SaleID                string    `json:"sale_id" gorm:"column:sale_id"`
	ProductID             string    `json:"product_id" gorm:"column:product_id"`
	ProductName           string    `json:"product_name" gorm:"column:product_name"`
	SKU                   string    `json:"sku,omitempty" gorm:"column:sku"`
	UnitType              string    `json:"unit_type" gorm:"column:unit_type"`
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

func (Sale) TableName() string     { return "sales" }
func (SaleItem) TableName() string { return "sale_items" }

type CreateSaleRequest struct {
	PaymentMethod string                  `json:"payment_method"`
	PaidAmount    float64                 `json:"paid_amount"`
	DiscountBill  float64                 `json:"discount_bill,omitempty"`
	VATIncluded   *bool                   `json:"vat_included,omitempty"`
	VATPercent    *float64                `json:"vat_percent,omitempty"`
	Note          string                  `json:"note"`
	CustomerID    string                  `json:"customer_id"`
	Items         []CreateSaleItemRequest `json:"items"`
}

type CreateSaleItemRequest struct {
	ProductID     string   `json:"product_id"`
	Quantity      int      `json:"quantity"`
	DiscountType  string   `json:"discount_type"`
	DiscountValue *float64 `json:"discount_value"`
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
