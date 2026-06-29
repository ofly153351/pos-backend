package document

import "time"

// ── Request DTOs ──────────────────────────────────────────────────────────────

type CreateDocumentRequest struct {
	Type         DocumentType              `json:"type"`
	CustomerID       string `json:"customer_id"`
	CustomerNameOverride    string `json:"customer_name,omitempty"`
	CustomerAddressOverride string `json:"customer_address,omitempty"`
	CustomerPhoneOverride   string `json:"customer_phone,omitempty"`
	DocumentDate string                    `json:"document_date"` // YYYY-MM-DD
	DueDate      *string                   `json:"due_date,omitempty"`
	ValidUntil      *string                   `json:"valid_until,omitempty"`
	DeliveryDate    *string                   `json:"delivery_date,omitempty"`    // YYYY-MM-DD
	DeliveryAddress string                    `json:"delivery_address"`
	DeliveryContact string                    `json:"delivery_contact"`
	DeliveryPhone   string                    `json:"delivery_phone"`
	SalesZone       string                    `json:"sales_zone"`
	SalespersonName string                    `json:"salesperson_name"`
	InvoiceRefNo     string                    `json:"invoice_ref_no"`
	SourceDocumentID *string                   `json:"source_document_id,omitempty"`
	PORefNo         string                    `json:"po_ref_no"`
	ShippingFee     float64                   `json:"shipping_fee"`
	CreditTermDays  int                       `json:"credit_term_days"`
	Items           []CreateDocumentItemInput `json:"items"`
	VatRate      float64                   `json:"vat_rate"`
	Notes        *string                   `json:"notes,omitempty"`
}

type CreateDocumentItemInput struct {
	ProductID     *string `json:"product_id,omitempty"`
	Description   string  `json:"description"`
	Unit          string  `json:"unit"`
	Quantity      float64 `json:"quantity"`
	UnitPrice     float64 `json:"unit_price"`
	DiscountType  string  `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`
}

type UpdateStatusRequest struct {
	Status DocumentStatus `json:"status"`
}

type UpdatePaymentStatusRequest struct {
	PaymentStatus PaymentStatus `json:"payment_status"`
}

type ConvertRequest struct {
	TargetType DocumentType `json:"target_type"`
}

// RelatedDoc is a lightweight projection of a document in the same conversion
// family (linked through source_document_id), used to render the lineage timeline.
type RelatedDoc struct {
	ID               string         `json:"id"`
	DocumentNo       string         `json:"document_no"`
	DocumentNoFull   string         `json:"document_no_full"`
	Type             DocumentType   `json:"type"`
	Status           DocumentStatus `json:"status"`
	PaymentStatus    PaymentStatus  `json:"payment_status"`
	DocumentDate     time.Time      `json:"document_date"`
	TotalAmount      float64        `json:"total_amount"`
	SourceDocumentID *string        `json:"source_document_id,omitempty"`
}

type BulkActionRequest struct {
	IDs    []string        `json:"ids"`
	Action string          `json:"action"` // DELETE | SET_STATUS
	Status *DocumentStatus `json:"status,omitempty"`
}

type ListQuery struct {
	StoreID       string
	Page          int
	Limit         int
	Search        string
	Type          string
	Status        string
	PaymentStatus string
	CustomerID    string
	StaffID       string
	DateFrom      string
	DateTo        string
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type DocumentStats struct {
	Total          int64 `json:"total"`
	PendingPayment int64 `json:"pending_payment"`
	Overdue        int64 `json:"overdue"`
	Paid           int64 `json:"paid"`
}

type DocumentListItem struct {
	ID               string         `json:"id"`
	DocumentNo       string         `json:"document_no"`
	DocumentNoFull   string         `json:"document_no_full"`
	Type             DocumentType   `json:"type"`
	Status           DocumentStatus `json:"status"`
	PaymentStatus    PaymentStatus  `json:"payment_status"`
	CustomerName     string         `json:"customer_name"`
	StaffName        string         `json:"staff_name"`
	DocumentDate     time.Time      `json:"document_date"`
	DueDate          *time.Time     `json:"due_date,omitempty"`
	TotalAmount      float64        `json:"total_amount"`
	SourceDocumentID *string        `json:"source_document_id,omitempty"`
}

type DocumentListResponse struct {
	Items []DocumentListItem `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Stats DocumentStats      `json:"stats"`
}
