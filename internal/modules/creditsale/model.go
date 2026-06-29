package creditsale

// Status values stored on the credit_sales row. "overdue" is NOT stored — it is
// derived at read time by comparing due_date to now (a past due_date with an
// outstanding balance), mirroring the frontend computeStatus precedence.
const (
	StatusPending   = "pending"
	StatusPartial   = "partial"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
	StatusOverdue   = "overdue" // derived only, never persisted
)

const (
	creditPaymentMethod = "credit" // payment_method tag on the underlying sales row
	typeCredit          = "credit"
	typeLoan            = "loan"
)

// CreditSaleItem is a read projection of the underlying sale's sale_items.
type CreditSaleItem struct {
	SaleID      string  `json:"-" gorm:"column:sale_id"`
	ProductID   string  `json:"product_id" gorm:"column:product_id"`
	ProductName string  `json:"product_name" gorm:"column:product_name"`
	Unit        string  `json:"unit" gorm:"column:unit"`
	Price       float64 `json:"price" gorm:"column:price"`
	Quantity    int     `json:"quantity" gorm:"column:quantity"`
	Total       float64 `json:"total" gorm:"column:total"`
	// ReturnedQty is how many of this line's units have already been returned to stock
	// (loan returns). Derived from RETURN stock_movements; not persisted on the item.
	ReturnedQty int `json:"returned_qty" gorm:"-"`
}

// CreditPayment is one collection against a receivable (the payment-history timeline).
type CreditPayment struct {
	ID           string  `json:"id" gorm:"column:id"`
	CreditSaleID string  `json:"-" gorm:"column:credit_sale_id"`
	StoreID      string  `json:"-" gorm:"column:store_id"`
	Amount       float64 `json:"amount" gorm:"column:amount"`
	Method       string  `json:"method" gorm:"column:method"`
	Note         string  `json:"note" gorm:"column:note"`
	PaidAt       string  `json:"paid_at" gorm:"column:paid_at"`
	CreatedBy    string  `json:"-" gorm:"column:created_by"`
}

// CreditSale is the receivable header + joined customer + items + payments.
type CreditSale struct {
	ID              string  `json:"id" gorm:"column:id"`
	StoreID         string  `json:"-" gorm:"column:store_id"`
	SaleID          string  `json:"sale_id" gorm:"column:sale_id"`
	DocumentNumber  string  `json:"document_number" gorm:"column:document_number"`
	Type            string  `json:"type" gorm:"column:type"`
	CustomerID      string  `json:"customer_id" gorm:"column:customer_id"`
	CustomerName    string  `json:"customer_name" gorm:"column:customer_name"`
	CustomerPhone   string  `json:"customer_phone" gorm:"column:customer_phone"`
	TotalAmount     float64 `json:"total_amount" gorm:"column:total_amount"`
	PaidAmount      float64 `json:"paid_amount" gorm:"column:paid_amount"`
	RemainingAmount float64 `json:"remaining_amount" gorm:"column:remaining_amount"`
	Status          string  `json:"status" gorm:"column:status"`
	DueDate         string  `json:"due_date" gorm:"column:due_date"`
	Note            string  `json:"note" gorm:"column:note"`
	CreatedBy       string  `json:"created_by" gorm:"column:created_by"`
	CreatedAt       string  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       string  `json:"updated_at" gorm:"column:updated_at"`

	Items    []CreditSaleItem `json:"items" gorm:"-"`
	Payments []CreditPayment  `json:"payments" gorm:"-"`
}

// ── Requests ────────────────────────────────────────────────────────────────

type CreateCreditItem struct {
	ProductID     string   `json:"product_id"`
	Quantity      int      `json:"quantity"`
	DiscountType  string   `json:"discount_type"`
	DiscountValue *float64 `json:"discount_value"`
}

type CreateCreditSaleRequest struct {
	Type           string             `json:"type"`
	CustomerID     string             `json:"customer_id"`
	DueDate        string             `json:"due_date"`
	Note           string             `json:"note"`
	DownPayment    float64            `json:"down_payment"`
	IdempotencyKey string             `json:"-"` // set from the Idempotency-Key header; dedupes a retried submit
	Items          []CreateCreditItem `json:"items"`

	// Optional pricing controls. When nil the underlying sale falls back to the
	// store's POS defaults (e.g. store VAT rate). These let the credit-sale form —
	// and POS "open credit bill" — carry the same discount/VAT/location intent as a
	// normal sale so the receivable total matches what the cashier saw.
	BillDiscount  *float64 `json:"bill_discount"`  // explicit end-of-bill discount
	VATPercent    *float64 `json:"vat_percent"`    // VAT rate %; nil → store POS default
	VATIncluded   *bool    `json:"vat_included"`   // true = prices already include VAT; nil → store default
	LocationID    string   `json:"location_id"`    // sale-point location (else store default)
	PromoDiscount *float64 `json:"promo_discount"` // promotion-derived discount (server-verified)
	PromotionIDs  []string `json:"promotion_ids"`  // applied active promotion ids
}

type AddPaymentRequest struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
	Note   string  `json:"note"`
}

// ReturnGoodsItem is one product+quantity being returned against a loan.
type ReturnGoodsItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// ReturnGoodsRequest records the return of borrowed goods (loan type only): the listed
// quantities are restocked at their original sale location and the receivable is settled
// by the value of the returned goods.
type ReturnGoodsRequest struct {
	Items []ReturnGoodsItem `json:"items"`
	Note  string            `json:"note"`
}

// DebtSummary powers the credit-sales KPI cards.
type DebtSummary struct {
	TotalOutstanding float64 `json:"total_outstanding"`
	OpenCount        int64   `json:"open_count"`
	OverdueCount     int64   `json:"overdue_count"`
	OverdueAmount    float64 `json:"overdue_amount"`
}

// AgingBucket is one row of the aging report: how many receivables fall in a given
// days-past-due range and the total outstanding for that range.
type AgingBucket struct {
	Label  string  `json:"label"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

// AgingCustomerEntry is one customer's total outstanding, bucketed by worst overdue.
type AgingCustomerEntry struct {
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	Outstanding  float64 `json:"outstanding"`
	OldestBucket string  `json:"oldest_bucket"`
	DaysOverdue  int     `json:"days_overdue"`
}

// AgingSummary is the full aging breakdown for a store.
type AgingSummary struct {
	Buckets   []AgingBucket        `json:"buckets"`
	Customers []AgingCustomerEntry `json:"customers"`
	Total     float64              `json:"total"`
}

// StatementContext is everything the statement PDF renderer needs: store + customer
// header info plus all of the customer's non-cancelled credit sales (with payments).
type StatementContext struct {
	StoreName    string
	StoreAddress string
	StoreTaxID   string
	CustomerName string
	CustomerAddr string
	DocumentNo   string
	Sales        []CreditSale
}
