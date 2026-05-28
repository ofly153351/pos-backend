package document

import "time"

// ── Request DTOs ──────────────────────────────────────────────────────────────

type CreateDocumentRequest struct {
	Type         DocumentType              `json:"type"`
	CustomerID   string                    `json:"customer_id"`
	DocumentDate string                    `json:"document_date"` // YYYY-MM-DD
	DueDate      *string                   `json:"due_date,omitempty"`
	Items        []CreateDocumentItemInput `json:"items"`
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
	ID             string         `json:"id"`
	DocumentNo     string         `json:"document_no"`
	DocumentNoFull string         `json:"document_no_full"`
	Type           DocumentType   `json:"type"`
	Status         DocumentStatus `json:"status"`
	PaymentStatus  PaymentStatus  `json:"payment_status"`
	CustomerName   string         `json:"customer_name"`
	StaffName      string         `json:"staff_name"`
	DocumentDate   time.Time      `json:"document_date"`
	DueDate        *time.Time     `json:"due_date,omitempty"`
	TotalAmount    float64        `json:"total_amount"`
}

type DocumentListResponse struct {
	Items []DocumentListItem `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Stats DocumentStats      `json:"stats"`
}
