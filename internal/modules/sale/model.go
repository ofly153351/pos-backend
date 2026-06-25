package sale

import "time"

type Sale struct {
	ID                     string     `json:"id" gorm:"column:id;primaryKey"`
	StoreID                string     `json:"store_id" gorm:"column:store_id"`
	LocationID             string     `json:"location_id,omitempty" gorm:"column:location_id"` // Phase W4B — effective sale-point location deducted from
	SaleNumber             string     `json:"sale_number" gorm:"column:sale_number"`
	CashierUserID          string     `json:"cashier_user_id" gorm:"column:cashier_user_id"`
	CashierName            string     `json:"cashier_name,omitempty" gorm:"column:cashier_name"`
	Status                 string     `json:"status" gorm:"column:status"`
	PaymentMethod          string     `json:"payment_method" gorm:"column:payment_method"`
	Note                   string     `json:"note,omitempty" gorm:"column:note"`
	CustomerID             string     `json:"customer_id,omitempty" gorm:"column:customer_id"`
	CustomerName           string     `json:"customer_name,omitempty" gorm:"column:customer_name"`
	CustomerPhone          string     `json:"customer_phone,omitempty" gorm:"column:customer_phone"`
	CustomerTaxID          string     `json:"customer_tax_id,omitempty" gorm:"column:customer_tax_id"`
	CustomerBranch         string     `json:"customer_branch,omitempty" gorm:"column:customer_branch"`
	StoreName              string     `json:"store_name,omitempty" gorm:"column:store_name"`
	StoreAddress           string     `json:"store_address,omitempty" gorm:"column:store_address"`
	StorePhone             string     `json:"store_phone,omitempty" gorm:"column:store_phone"`
	StorePromptPayID       string     `json:"store_promptpay_id,omitempty" gorm:"column:store_promptpay_id"`
	StoreTaxID             string     `json:"store_tax_id,omitempty" gorm:"column:store_tax_id"`
	StoreLogoURL           string     `json:"store_logo_url,omitempty" gorm:"column:store_logo_url"`
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
	VoidedAt               *time.Time `json:"voided_at,omitempty" gorm:"column:voided_at"`
	VoidedBy               string     `json:"voided_by,omitempty" gorm:"column:voided_by"`
	VoidReason             string     `json:"void_reason,omitempty" gorm:"column:void_reason"`
	VoidType               string     `json:"void_type,omitempty" gorm:"column:void_type"`
	IdempotencyKey         string     `json:"-" gorm:"column:idempotency_key"`     // Phase W4B
	RequestFingerprint     string     `json:"-" gorm:"column:request_fingerprint"` // Phase W4B
	Items                  []SaleItem   `json:"items,omitempty" gorm:"foreignKey:SaleID;references:ID"`
	Returns                []SaleReturn `json:"returns,omitempty" gorm:"-"` // migration 052 — loaded in GetByID
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
	UnitCost              float64   `json:"unit_cost" gorm:"column:unit_cost"`
	DiscountType          string    `json:"discount_type,omitempty" gorm:"column:discount_type"`
	DiscountValue         *float64  `json:"discount_value,omitempty" gorm:"column:discount_value"`
	DiscountAmountPerUnit float64   `json:"discount_amount_per_unit" gorm:"column:discount_amount_per_unit"`
	LineSubtotal          float64   `json:"line_subtotal" gorm:"column:line_subtotal"`
	LineDiscountTotal     float64   `json:"line_discount_total" gorm:"column:line_discount_total"`
	LineTotal             float64   `json:"line_total" gorm:"column:line_total"`
	ReturnedQuantity      int       `json:"returned_quantity" gorm:"column:returned_quantity"` // migration 052
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
}

// SaleReturn is a partial-or-full return recorded against a sale. The original
// sale is never voided; status moves to partially_returned / fully_returned.
type SaleReturn struct {
	ID           string           `json:"id" gorm:"column:id;primaryKey"`
	StoreID      string           `json:"store_id" gorm:"column:store_id"`
	SaleID       string           `json:"sale_id" gorm:"column:sale_id"`
	ReturnNumber string           `json:"return_number" gorm:"column:return_number"`
	RefundMethod string           `json:"refund_method" gorm:"column:refund_method"`
	RefundAmount float64          `json:"refund_amount" gorm:"column:refund_amount"`
	Reason       string           `json:"reason,omitempty" gorm:"column:reason"`
	CreatedBy    string           `json:"created_by" gorm:"column:created_by"`
	CreatedByName string          `json:"created_by_name,omitempty" gorm:"column:created_by_name"`
	CreatedAt    time.Time        `json:"created_at" gorm:"column:created_at"`
	Items        []SaleReturnItem `json:"items,omitempty" gorm:"foreignKey:ReturnID;references:ID"`
}

type SaleReturnItem struct {
	ID          string  `json:"id" gorm:"column:id;primaryKey"`
	ReturnID    string  `json:"return_id" gorm:"column:return_id"`
	SaleItemID  string  `json:"sale_item_id" gorm:"column:sale_item_id"`
	ProductID   string  `json:"product_id" gorm:"column:product_id"`
	ProductName string  `json:"product_name,omitempty" gorm:"column:product_name"`
	SKU         string  `json:"sku,omitempty" gorm:"column:sku"`
	Quantity    int     `json:"quantity" gorm:"column:quantity"`
	UnitPrice   float64 `json:"unit_price" gorm:"column:unit_price"`
	LineRefund  float64 `json:"line_refund" gorm:"column:line_refund"`
}

func (Sale) TableName() string           { return "sales" }
func (SaleItem) TableName() string       { return "sale_items" }
func (SaleReturn) TableName() string     { return "sale_returns" }
func (SaleReturnItem) TableName() string { return "sale_return_items" }

// CreateReturnRequest is the partial-return payload. Items carry the line id (or
// product id as fallback) and the quantity to return; the server authoritatively
// computes the refund from the original sale_items, never trusting client amounts.
type CreateReturnRequest struct {
	RefundMethod string              `json:"refund_method"`
	Reason       string              `json:"reason"`
	Items        []ReturnItemRequest `json:"items"`
}

type ReturnItemRequest struct {
	SaleItemID string `json:"sale_item_id"`
	ProductID  string `json:"product_id"`
	Quantity   int    `json:"quantity"`
}

type CreateSaleRequest struct {
	PaymentMethod  string                  `json:"payment_method"`
	LocationID     string                  `json:"location_id,omitempty"` // Phase W4B — explicit POS sale-point location (else store default)
	IdempotencyKey string                  `json:"-"`                     // Phase W4B — set from the Idempotency-Key header
	PaidAmount     float64                 `json:"paid_amount"`
	DiscountBill   float64                 `json:"discount_bill,omitempty"`   // deprecated fallback (treated as manual)
	ManualDiscount *float64                `json:"manual_discount,omitempty"` // explicit cashier bill discount
	PromoDiscount  *float64                `json:"promo_discount,omitempty"`  // promotion-derived discount (server-verified)
	PromotionIDs   []string                `json:"promotion_ids,omitempty"`   // applied active promotion ids
	VATIncluded    *bool                   `json:"vat_included,omitempty"`
	VATPercent     *float64                `json:"vat_percent,omitempty"`
	Note           string                  `json:"note"`
	CustomerID     string                  `json:"customer_id"`
	Items          []CreateSaleItemRequest `json:"items"`
}

// DiscountInput carries the validated bill-discount inputs from the request into the
// repository, where the cart subtotal is known. ManualDiscount/PromoDiscount are
// pointers so an absent (nil) split falls back to the deprecated LegacyBill field.
type DiscountInput struct {
	ManualDiscount *float64
	PromoDiscount  *float64
	PromotionIDs   []string
	LegacyBill     float64 // discount_bill fallback (treated as manual)
	IsElevated     bool    // actor is owner/manager/platform_admin for this store
}

// appliedPromoResult reports which promotions the server actually honored on a sale
// and the total verified promo discount, so the sale-create transaction can record
// promotion usage (promotion_usages + promotions.usage_count/discount_given_total).
type appliedPromoResult struct {
	VerifiedDiscount float64
	PromotionIDs     []string
}

type VoidSaleRequest struct {
	Reason string `json:"reason"`
	Type   string `json:"type"` // "void" or "return"
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
	IsActive            bool
	BasePrice           float64
	CostPrice           float64
	SpecialPrice        *float64
	SpecialPriceStartAt *time.Time
	SpecialPriceEndAt   *time.Time
}
