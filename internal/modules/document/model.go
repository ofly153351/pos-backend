package document

import "time"

type DocumentType   string
type DocumentStatus string
type PaymentStatus  string

const (
	TypeInvoice    DocumentType = "INVOICE"
	TypeReceipt    DocumentType = "RECEIPT"
	TypeTaxInvoice DocumentType = "TAX_INVOICE"
	TypeQuotation  DocumentType = "QUOTATION"
	TypeBill       DocumentType = "BILL"
	TypeCreditNote DocumentType = "CREDIT_NOTE"
)

const (
	StatusDraft     DocumentStatus = "DRAFT"
	StatusPending   DocumentStatus = "PENDING"
	StatusOverdue   DocumentStatus = "OVERDUE"
	StatusCompleted DocumentStatus = "COMPLETED"
	StatusCancelled DocumentStatus = "CANCELLED"
)

const (
	PaymentUnpaid  PaymentStatus = "UNPAID"
	PaymentPartial PaymentStatus = "PARTIAL"
	PaymentPaid    PaymentStatus = "PAID"
)

type Document struct {
	ID             string         `gorm:"primaryKey;type:varchar(30)" json:"id"`
	StoreID        string         `gorm:"not null;index" json:"store_id"`
	DocumentNo     string         `gorm:"not null;uniqueIndex" json:"document_no"`
	DocumentNoFull string         `gorm:"not null" json:"document_no_full"`
	Type           DocumentType   `gorm:"not null;type:varchar(20)" json:"type"`
	Status         DocumentStatus `gorm:"not null;type:varchar(20);default:'PENDING'" json:"status"`
	PaymentStatus  PaymentStatus  `gorm:"not null;type:varchar(20);default:'UNPAID'" json:"payment_status"`
	CustomerID     string         `gorm:"not null" json:"customer_id"`
	CustomerName   string         `gorm:"not null" json:"customer_name"`
	CustomerTaxID  *string        `json:"customer_tax_id,omitempty"`
	StaffID        string         `gorm:"not null" json:"staff_id"`
	StaffName      string         `gorm:"not null" json:"staff_name"`
	DocumentDate   time.Time      `gorm:"not null" json:"document_date"`
	DueDate        *time.Time     `json:"due_date,omitempty"`
	Subtotal       float64        `gorm:"not null;default:0" json:"subtotal"`
	VatRate        float64        `gorm:"not null;default:0" json:"vat_rate"`
	VatAmount      float64        `gorm:"not null;default:0" json:"vat_amount"`
	TotalAmount    float64        `gorm:"not null;default:0" json:"total_amount"`
	Notes          *string        `json:"notes,omitempty"`
	Items          []DocumentItem `gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	CreatedBy      string         `gorm:"not null" json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type DocumentItem struct {
	ID            string  `gorm:"primaryKey;type:varchar(30)" json:"id"`
	DocumentID    string  `gorm:"not null;index" json:"document_id"`
	ProductID     *string `json:"product_id,omitempty"`
	Description   string  `gorm:"not null" json:"description"`
	Quantity      float64 `gorm:"not null;default:1" json:"quantity"`
	UnitPrice     float64 `gorm:"not null;default:0" json:"unit_price"`
	DiscountType  string  `gorm:"type:varchar(10);default:''" json:"discount_type"`
	DiscountValue float64 `gorm:"not null;default:0" json:"discount_value"`
	Amount        float64 `gorm:"not null;default:0" json:"amount"`
}

func (Document) TableName() string     { return "documents" }
func (DocumentItem) TableName() string { return "document_items" }
