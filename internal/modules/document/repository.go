package document

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	List(q ListQuery) ([]DocumentListItem, int64, DocumentStats, error)
	FindByID(id string) (*Document, error)
	Create(doc *Document) error
	UpdateStatus(id string, status DocumentStatus) error
	MarkPaid(id string) error
	Delete(id string) error
	BulkDelete(ids []string) error
	BulkSetStatus(ids []string, status DocumentStatus) error
	NextSeq(storeID string, docType DocumentType) (int64, error)
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db} }

func (r *repository) List(q ListQuery) ([]DocumentListItem, int64, DocumentStats, error) {
	base := r.db.Model(&Document{})

	if q.StoreID != "" {
		base = base.Where("store_id = ?", q.StoreID)
	}
	if q.Search != "" {
		like := "%" + strings.ToLower(q.Search) + "%"
		base = base.Where("LOWER(document_no) LIKE ? OR LOWER(customer_name) LIKE ?", like, like)
	}
	if q.Type != "" {
		base = base.Where("type = ?", q.Type)
	}
	if q.Status != "" {
		base = base.Where("status = ?", q.Status)
	}
	if q.PaymentStatus != "" {
		base = base.Where("payment_status = ?", q.PaymentStatus)
	}
	if q.CustomerID != "" {
		base = base.Where("customer_id = ?", q.CustomerID)
	}
	if q.StaffID != "" {
		base = base.Where("staff_id = ?", q.StaffID)
	}
	if q.DateFrom != "" {
		base = base.Where("document_date >= ?", q.DateFrom)
	}
	if q.DateTo != "" {
		base = base.Where("document_date <= ?", q.DateTo)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, DocumentStats{}, err
	}

	// Stats with same filters
	var stats DocumentStats
	if err := base.Select(`
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE status IN ('PENDING','OVERDUE') AND payment_status != 'PAID') AS pending_payment,
		COUNT(*) FILTER (WHERE status = 'OVERDUE') AS overdue,
		COUNT(*) FILTER (WHERE payment_status = 'PAID') AS paid
	`).Scan(&stats).Error; err != nil {
		return nil, 0, DocumentStats{}, err
	}

	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	var rows []DocumentListItem
	err := base.
		Select("id, document_no, document_no_full, type, status, payment_status, customer_name, staff_name, document_date, due_date, total_amount, source_document_id").
		Order("document_date DESC, created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&rows).Error

	return rows, total, stats, err
}

func (r *repository) FindByID(id string) (*Document, error) {
	var doc Document
	err := r.db.Preload("Items").First(&doc, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *repository) Create(doc *Document) error {
	return r.db.Create(doc).Error
}

func (r *repository) UpdateStatus(id string, status DocumentStatus) error {
	return r.db.Model(&Document{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *repository) MarkPaid(id string) error {
	return r.db.Model(&Document{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         StatusCompleted,
			"payment_status": PaymentPaid,
			"updated_at":     time.Now(),
		}).Error
}

func (r *repository) Delete(id string) error {
	return r.db.Delete(&Document{}, "id = ?", id).Error
}

func (r *repository) BulkDelete(ids []string) error {
	return r.db.Delete(&Document{}, "id IN ?", ids).Error
}

func (r *repository) BulkSetStatus(ids []string, status DocumentStatus) error {
	return r.db.Model(&Document{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// NextSeq returns the next sequence number for a given store+type combo in the current month.
func (r *repository) NextSeq(storeID string, docType DocumentType) (int64, error) {
	now := time.Now()
	prefix := fmt.Sprintf("%s-%02d%02d", typePrefix(docType), now.Year()%100, now.Month())
	var count int64
	r.db.Model(&Document{}).
		Where("store_id = ? AND document_no LIKE ?", storeID, prefix+"-%").
		Count(&count)
	return count + 1, nil
}

func typePrefix(t DocumentType) string {
	switch t {
	case TypeInvoice:
		return "INV"
	case TypeReceipt:
		return "RCT"
	case TypeTaxInvoice:
		return "TAX"
	case TypeQuotation:
		return "QUO"
	case TypeBill:
		return "BILL"
	case TypeCreditNote:
		return "CN"
	case TypeDeliveryOrder:
		return "DO"
	default:
		return "DOC"
	}
}
