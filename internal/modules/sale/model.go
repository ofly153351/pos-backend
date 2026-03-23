package sale

import "time"

type Sale struct {
	ID             string     `json:"id"`
	StoreID        string     `json:"store_id"`
	SaleNumber     string     `json:"sale_number"`
	CashierUserID  string     `json:"cashier_user_id"`
	Status         string     `json:"status"`
	PaymentMethod  string     `json:"payment_method"`
	Note           string     `json:"note,omitempty"`
	TotalItems     int        `json:"total_items"`
	SubtotalAmount float64    `json:"subtotal_amount"`
	DiscountAmount float64    `json:"discount_amount"`
	TotalAmount    float64    `json:"total_amount"`
	PaidAmount     float64    `json:"paid_amount"`
	ChangeAmount   float64    `json:"change_amount"`
	SoldAt         time.Time  `json:"sold_at"`
	CreatedAt      time.Time  `json:"created_at"`
	Items          []SaleItem `json:"items,omitempty"`
}

type SaleItem struct {
	ID                    string    `json:"id"`
	SaleID                string    `json:"sale_id"`
	ProductID             string    `json:"product_id"`
	ProductName           string    `json:"product_name"`
	SKU                   string    `json:"sku,omitempty"`
	UnitType              string    `json:"unit_type"`
	Quantity              int       `json:"quantity"`
	UnitPrice             float64   `json:"unit_price"`
	DiscountType          string    `json:"discount_type,omitempty"`
	DiscountValue         *float64  `json:"discount_value,omitempty"`
	DiscountAmountPerUnit float64   `json:"discount_amount_per_unit"`
	LineSubtotal          float64   `json:"line_subtotal"`
	LineDiscountTotal     float64   `json:"line_discount_total"`
	LineTotal             float64   `json:"line_total"`
	CreatedAt             time.Time `json:"created_at"`
}

type CreateSaleRequest struct {
	PaymentMethod string                  `json:"payment_method"`
	PaidAmount    float64                 `json:"paid_amount"`
	Note          string                  `json:"note"`
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
