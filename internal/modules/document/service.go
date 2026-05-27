package document

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pos-backend/internal/idgen"
	"pos-backend/internal/modules/auth"

	"gorm.io/gorm"
)

var (
	ErrNotFound     = errors.New("document not found")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrNoItems      = errors.New("document must have at least one item")
	ErrBadAction    = errors.New("unknown bulk action")
)

const (
	PrefixDocument     = "doc"
	PrefixDocumentItem = "doci"
)

type Service struct {
	repo Repository
	db   *gorm.DB
}

func NewService(repo Repository, db *gorm.DB) Service {
	return Service{repo: repo, db: db}
}

func (s Service) ListDocuments(ctx context.Context, actor auth.Claims, q ListQuery) (DocumentListResponse, error) {
	if err := s.ensureAccess(actor, q.StoreID); err != nil {
		return DocumentListResponse{}, err
	}
	items, total, stats, err := s.repo.List(q)
	if err != nil {
		return DocumentListResponse{}, err
	}
	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 {
		limit = 20
	}
	return DocumentListResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
		Stats: stats,
	}, nil
}

func (s Service) GetDocument(ctx context.Context, actor auth.Claims, storeID, id string) (*Document, error) {
	if err := s.ensureAccess(actor, storeID); err != nil {
		return nil, err
	}
	doc, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if doc.StoreID != storeID {
		return nil, ErrForbidden
	}
	return doc, nil
}

func (s Service) CreateDocument(ctx context.Context, actor auth.Claims, storeID string, req CreateDocumentRequest) (*Document, error) {
	if err := s.ensureAccess(actor, storeID); err != nil {
		return nil, err
	}
	if len(req.Items) == 0 {
		return nil, ErrNoItems
	}

	// Resolve customer
	var cust struct {
		FullName string
		TaxID    *string
	}
	if err := s.db.Raw(
		"SELECT full_name, tax_id FROM customers WHERE id = ? AND store_id = ?",
		req.CustomerID, storeID,
	).Scan(&cust).Error; err != nil || cust.FullName == "" {
		return nil, fmt.Errorf("customer not found: %w", ErrInvalidInput)
	}

	docDate, err := time.Parse("2006-01-02", req.DocumentDate)
	if err != nil {
		return nil, fmt.Errorf("invalid document_date: %w", ErrInvalidInput)
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date: %w", ErrInvalidInput)
		}
		dueDate = &t
	}

	// Generate document number
	seq, _ := s.repo.NextSeq(storeID, req.Type)
	prefix := typePrefix(req.Type)
	now := time.Now()
	buddhistYear := now.Year() + 543
	docNo := fmt.Sprintf("%s-%02d%02d-%04d", prefix, now.Year()%100, int(now.Month()), seq)
	docNoFull := fmt.Sprintf("%s/%d/%02d/%04d", prefix, buddhistYear, int(now.Month()), seq)

	// Build items + totals
	var subtotal float64
	items := make([]DocumentItem, 0, len(req.Items))
	for _, inp := range req.Items {
		lineAmt := inp.Quantity * inp.UnitPrice
		switch inp.DiscountType {
		case "PERCENT":
			lineAmt -= lineAmt * inp.DiscountValue / 100
		case "AMOUNT":
			lineAmt -= inp.DiscountValue
		}
		if lineAmt < 0 {
			lineAmt = 0
		}
		subtotal += lineAmt
		items = append(items, DocumentItem{
			ID:            idgen.Generate(PrefixDocumentItem),
			Description:   inp.Description,
			ProductID:     inp.ProductID,
			Quantity:      inp.Quantity,
			UnitPrice:     inp.UnitPrice,
			DiscountType:  inp.DiscountType,
			DiscountValue: inp.DiscountValue,
			Amount:        lineAmt,
		})
	}
	vatAmount := subtotal * req.VatRate / 100
	totalAmount := subtotal + vatAmount

	doc := &Document{
		ID:             idgen.Generate(PrefixDocument),
		StoreID:        storeID,
		DocumentNo:     docNo,
		DocumentNoFull: docNoFull,
		Type:           req.Type,
		Status:         StatusPending,
		PaymentStatus:  PaymentUnpaid,
		CustomerID:     req.CustomerID,
		CustomerName:   cust.FullName,
		CustomerTaxID:  cust.TaxID,
		StaffID:        actor.UserID,
		StaffName:      actor.Name,
		DocumentDate:   docDate,
		DueDate:        dueDate,
		Subtotal:       subtotal,
		VatRate:        req.VatRate,
		VatAmount:      vatAmount,
		TotalAmount:    totalAmount,
		Notes:          req.Notes,
		Items:          items,
		CreatedBy:      actor.UserID,
	}

	if err := s.repo.Create(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s Service) UpdateDocumentStatus(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateStatusRequest) error {
	if _, err := s.GetDocument(ctx, actor, storeID, id); err != nil {
		return err
	}
	return s.repo.UpdateStatus(id, req.Status)
}

func (s Service) DeleteDocument(ctx context.Context, actor auth.Claims, storeID, id string) error {
	if _, err := s.GetDocument(ctx, actor, storeID, id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s Service) BulkAction(ctx context.Context, actor auth.Claims, storeID string, req BulkActionRequest) error {
	if err := s.ensureAccess(actor, storeID); err != nil {
		return err
	}
	switch req.Action {
	case "DELETE":
		return s.repo.BulkDelete(req.IDs)
	case "SET_STATUS":
		if req.Status == nil {
			return ErrInvalidInput
		}
		return s.repo.BulkSetStatus(req.IDs, *req.Status)
	default:
		return ErrBadAction
	}
}

func (s Service) ensureAccess(actor auth.Claims, storeID string) error {
	if storeID == "" {
		return ErrInvalidInput
	}
	return nil
}
