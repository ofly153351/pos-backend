package customer

import "time"

type CustomerShippingAddress struct {
	ID                 string    `json:"id" gorm:"column:id;primaryKey"`
	CustomerID         string    `json:"customer_id" gorm:"column:customer_id"`
	Label              string    `json:"label" gorm:"column:label"`
	RecipientName      string    `json:"recipient_name" gorm:"column:recipient_name"`
	RecipientPhone     string    `json:"recipient_phone" gorm:"column:recipient_phone"`
	Address            string    `json:"address" gorm:"column:address"`
	SubDistrict        string    `json:"sub_district" gorm:"column:sub_district"`
	District           string    `json:"district" gorm:"column:district"`
	Province           string    `json:"province" gorm:"column:province"`
	PostalCode         string    `json:"postal_code" gorm:"column:postal_code"`
	Note               string    `json:"note" gorm:"column:note"`
	UseCustomerAddress bool      `json:"use_customer_address" gorm:"column:use_customer_address"`
	IsDefault          bool      `json:"is_default" gorm:"column:is_default"`
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (CustomerShippingAddress) TableName() string {
	return "customer_shipping_addresses"
}

type Customer struct {
	ID         string `json:"id" gorm:"column:id;primaryKey"`
	StoreID    string `json:"store_id" gorm:"column:store_id"`
	Level      int    `json:"level" gorm:"column:customer_level"`
	MemberCode string `json:"member_code,omitempty" gorm:"column:member_code"`
	FullName   string `json:"full_name" gorm:"column:full_name"`
	Phone      string `json:"phone,omitempty" gorm:"column:phone"`
	Email      string `json:"email,omitempty" gorm:"column:email"`
	Address    string `json:"address,omitempty" gorm:"column:address"`
	Note       string `json:"note,omitempty" gorm:"column:note"`
	TaxID      string `json:"tax_id,omitempty" gorm:"column:tax_id"`
	Branch     string `json:"branch,omitempty" gorm:"column:branch"`
	Points     int    `json:"points" gorm:"column:points"`
	// Shipping / delivery profile (Phase 3) — separate from the billing Address above.
	ShippingContact    string `json:"shipping_contact,omitempty" gorm:"column:shipping_contact"`
	ShippingPhone      string `json:"shipping_phone,omitempty" gorm:"column:shipping_phone"`
	ShippingAddress    string `json:"shipping_address,omitempty" gorm:"column:shipping_address"`
	ShippingProvince   string `json:"shipping_province,omitempty" gorm:"column:shipping_province"`
	ShippingDistrict   string `json:"shipping_district,omitempty" gorm:"column:shipping_district"`
	ShippingPostalCode string `json:"shipping_postal_code,omitempty" gorm:"column:shipping_postal_code"`
	DeliveryNote       string `json:"delivery_note,omitempty" gorm:"column:delivery_note"`
	// Multi-address shipping (Phase 4)
	ShippingAddresses []CustomerShippingAddress `json:"shipping_addresses" gorm:"foreignKey:CustomerID"`
	IsActive          bool                      `json:"is_active" gorm:"column:is_active"`
	CreatedAt         time.Time                 `json:"created_at" gorm:"column:created_at"`
	UpdatedAt         time.Time                 `json:"updated_at" gorm:"column:updated_at"`
}

// CustomerListItem augments a Customer with read-time aggregates from the sales
// table (lifetime purchase value + completed bill count). These are not stored
// columns; they are computed by the list query so the customers screen can show
// "ยอดซื้อสะสม" / "จำนวนบิล" without an extra round-trip per row.
type CustomerListItem struct {
	Customer
	TotalPurchase float64 `json:"total_purchase" gorm:"column:total_purchase"`
	TotalBills    int     `json:"total_bills" gorm:"column:total_bills"`
}

type LevelDiscount struct {
	StoreID         string    `json:"store_id" gorm:"column:store_id;primaryKey"`
	Level           int       `json:"level" gorm:"column:level;primaryKey"`
	DiscountPercent float64   `json:"discount_percent" gorm:"column:discount_percent"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at"`
}

type CreateCustomerRequest struct {
	Level              *int   `json:"level"`
	FullName           string `json:"full_name"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	Address            string `json:"address"`
	Note               string `json:"note"`
	TaxID              string `json:"tax_id"`
	Branch             string `json:"branch"`
	ShippingContact    string `json:"shipping_contact"`
	ShippingPhone      string `json:"shipping_phone"`
	ShippingAddress    string `json:"shipping_address"`
	ShippingProvince   string `json:"shipping_province"`
	ShippingDistrict   string `json:"shipping_district"`
	ShippingPostalCode string `json:"shipping_postal_code"`
	DeliveryNote       string `json:"delivery_note"`
	IsActive           *bool  `json:"is_active"`
}

type UpdateCustomerRequest struct {
	Level              *int    `json:"level"`
	FullName           *string `json:"full_name"`
	Phone              *string `json:"phone"`
	Email              *string `json:"email"`
	Address            *string `json:"address"`
	Note               *string `json:"note"`
	TaxID              *string `json:"tax_id"`
	Branch             *string `json:"branch"`
	ShippingContact    *string `json:"shipping_contact"`
	ShippingPhone      *string `json:"shipping_phone"`
	ShippingAddress    *string `json:"shipping_address"`
	ShippingProvince   *string `json:"shipping_province"`
	ShippingDistrict   *string `json:"shipping_district"`
	ShippingPostalCode *string `json:"shipping_postal_code"`
	DeliveryNote       *string `json:"delivery_note"`
	IsActive           *bool   `json:"is_active"`
}

type UpsertLevelDiscountRequest struct {
	DiscountPercent float64 `json:"discount_percent"`
}

type ShippingAddressRequest struct {
	Label              string `json:"label"`
	RecipientName      string `json:"recipient_name"`
	RecipientPhone     string `json:"recipient_phone"`
	Address            string `json:"address"`
	SubDistrict        string `json:"sub_district"`
	District           string `json:"district"`
	Province           string `json:"province"`
	PostalCode         string `json:"postal_code"`
	Note               string `json:"note"`
	UseCustomerAddress bool   `json:"use_customer_address"`
	IsDefault          bool   `json:"is_default"`
}

func (Customer) TableName() string {
	return "customers"
}

func (LevelDiscount) TableName() string {
	return "customer_level_discounts"
}
