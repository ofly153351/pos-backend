package parkedbill

import "time"

type ParkedBill struct {
	ID                     string           `json:"id" gorm:"column:id;primaryKey"`
	StoreID                string           `json:"store_id" gorm:"column:store_id"`
	CashierUserID          string           `json:"cashier_user_id" gorm:"column:cashier_user_id"`
	Label                  string           `json:"label" gorm:"column:label"`
	Note                   string           `json:"note,omitempty" gorm:"column:note"`
	BillDiscountAmount     float64          `json:"bill_discount_amount" gorm:"column:bill_discount_amount"`
	BillDiscountType       string           `json:"bill_discount_type" gorm:"column:bill_discount_type"`
	BillDiscountPercent    float64          `json:"bill_discount_percent" gorm:"column:bill_discount_percent"`
	CustomerID             string           `json:"customer_id,omitempty" gorm:"column:customer_id"`
	CustomerSettlementMode string           `json:"customer_settlement_mode" gorm:"column:customer_settlement_mode"`
	PaymentMethod          string           `json:"payment_method" gorm:"column:payment_method"`
	VATIncluded            bool             `json:"vat_included" gorm:"column:vat_included"`
	VATPercent             float64          `json:"vat_percent" gorm:"column:vat_percent"`
	CreatedAt              time.Time        `json:"created_at" gorm:"column:created_at"`
	Items                  []ParkedBillItem `json:"items,omitempty" gorm:"foreignKey:ParkedBillID;references:ID"`
}

type ParkedBillItem struct {
	ID            string   `json:"id" gorm:"column:id;primaryKey"`
	ParkedBillID  string   `json:"parked_bill_id" gorm:"column:parked_bill_id"`
	ProductID     string   `json:"product_id" gorm:"column:product_id"`
	ProductName   string   `json:"product_name" gorm:"column:product_name"`
	ProductSKU    string   `json:"product_sku,omitempty" gorm:"column:product_sku"`
	Price         float64  `json:"price" gorm:"column:price"`
	Quantity      int      `json:"quantity" gorm:"column:quantity"`
	DiscountType  string   `json:"discount_type,omitempty" gorm:"column:discount_type"`
	DiscountValue *float64 `json:"discount_value,omitempty" gorm:"column:discount_value"`
}

func (ParkedBill) TableName() string     { return "parked_bills" }
func (ParkedBillItem) TableName() string { return "parked_bill_items" }

type CreateParkedBillRequest struct {
	Label                  string                         `json:"label"`
	Note                   string                         `json:"note,omitempty"`
	BillDiscountAmount     float64                        `json:"bill_discount_amount"`
	BillDiscountType       string                         `json:"bill_discount_type"`
	BillDiscountPercent    float64                        `json:"bill_discount_percent"`
	CustomerID             string                         `json:"customer_id,omitempty"`
	CustomerSettlementMode string                         `json:"customer_settlement_mode"`
	PaymentMethod          string                         `json:"payment_method"`
	VATIncluded            *bool                          `json:"vat_included,omitempty"`
	VATPercent             *float64                       `json:"vat_percent,omitempty"`
	Items                  []CreateParkedBillItemRequest  `json:"items"`
}

type CreateParkedBillItemRequest struct {
	ProductID     string   `json:"product_id"`
	Quantity      int      `json:"quantity"`
	DiscountType  string   `json:"discount_type,omitempty"`
	DiscountValue *float64 `json:"discount_value,omitempty"`
}

type ParkedBillResponse struct {
	Bill  ParkedBill      `json:"bill"`
	Items []ParkedBillItem `json:"items"`
}
