package vat

type VATItem struct {
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	Qty             int     `json:"qty"`
	Price           float64 `json:"price"`
	DiscountPerUnit float64 `json:"discount_per_unit,omitempty"`
}

type CalculateVATRequest struct {
	Items        []VATItem `json:"items"`
	DiscountBill float64   `json:"discount_bill,omitempty"`
	VATPercent   float64   `json:"vat_percent,omitempty"`
	VATIncluded  *bool     `json:"vat_included,omitempty"`
}

type VATSummary struct {
	Subtotal      float64 `json:"subtotal"`
	DiscountItem  float64 `json:"discount_item"`
	DiscountBill  float64 `json:"discount_bill"`
	AfterDiscount float64 `json:"after_discount"`
	VATPercent    float64 `json:"vat_percent"`
	VATAmount     float64 `json:"vat_amount"`
	GrandTotal    float64 `json:"grand_total"`
	VATIncluded   bool    `json:"vat_included"`
}
