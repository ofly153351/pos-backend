package document

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"pos-backend/internal/idgen"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/dochtml"

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

	var store struct {
		Name    string `gorm:"column:name"`
		Address string `gorm:"column:address"`
		Phone   string `gorm:"column:phone"`
		Fax     string `gorm:"column:fax"`
		Email   string `gorm:"column:email"`
		Website string `gorm:"column:website"`
		TaxID   string `gorm:"column:tax_id"`
		LogoURL string `gorm:"column:logo_url"`
	}
	_ = s.db.Raw(
		"SELECT name, COALESCE(address,'') AS address, COALESCE(phone,'') AS phone, COALESCE(fax,'') AS fax, COALESCE(email,'') AS email, COALESCE(website,'') AS website, COALESCE(tax_id,'') AS tax_id, COALESCE(logo_url,'') AS logo_url FROM stores WHERE id = ?",
		storeID,
	).Scan(&store)
	doc.StoreName = store.Name
	doc.StoreAddress = store.Address
	doc.StorePhone = store.Phone
	doc.StoreFax = store.Fax
	doc.StoreEmail = store.Email
	doc.StoreWebsite = store.Website
	doc.StoreTaxID = store.TaxID
	doc.StoreLogoURL = store.LogoURL

	return doc, nil
}

func (s Service) CreateDocument(ctx context.Context, actor auth.Claims, storeID string, req CreateDocumentRequest) (*Document, error) {
	if err := s.ensureAccess(actor, storeID); err != nil {
		return nil, err
	}
	if len(req.Items) == 0 {
		return nil, ErrNoItems
	}

	// Resolve customer (snapshot name + address + phone for §86/4 compliance)
	var cust struct {
		FullName string
		Address  string
		Phone    string
	}
	if err := s.db.Raw(
		"SELECT full_name, COALESCE(address,'') AS address, COALESCE(phone,'') AS phone FROM customers WHERE id = ? AND store_id = ?",
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

	// Build items + totals (round to 2 dp to avoid float64 precision drift)
	round2 := func(v float64) float64 { return math.Round(v*100) / 100 }

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
		lineAmt = round2(lineAmt)
		subtotal += lineAmt
		unit := inp.Unit
		if unit == "" {
			unit = "ชิ้น"
		}
		items = append(items, DocumentItem{
			ID:            idgen.Generate(PrefixDocumentItem),
			Description:   inp.Description,
			Unit:          unit,
			ProductID:     inp.ProductID,
			Quantity:      inp.Quantity,
			UnitPrice:     inp.UnitPrice,
			DiscountType:  inp.DiscountType,
			DiscountValue: inp.DiscountValue,
			Amount:        lineAmt,
		})
	}
	subtotal = round2(subtotal)
	vatAmount := round2(subtotal * req.VatRate / 100)
	totalAmount := round2(subtotal + vatAmount)

	doc := &Document{
		ID:              idgen.Generate(PrefixDocument),
		StoreID:         storeID,
		DocumentNo:      docNo,
		DocumentNoFull:  docNoFull,
		Type:            req.Type,
		Status:          StatusPending,
		PaymentStatus:   PaymentUnpaid,
		CustomerID:      req.CustomerID,
		CustomerName:    cust.FullName,
		CustomerAddress: cust.Address,
		CustomerPhone:   cust.Phone,
		StaffID:         actor.UserID,
		StaffName:       actor.Name,
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

func (s Service) RenderDocumentPrint(ctx context.Context, actor auth.Claims, storeID, id string) (string, error) {
	doc, err := s.GetDocument(ctx, actor, storeID, id)
	if err != nil {
		return "", err
	}

	return dochtml.RenderDocumentHTML(toDocData(doc), dochtml.StoreInfo{
		Name:    doc.StoreName,
		Address: doc.StoreAddress,
		Phone:   doc.StorePhone,
		Fax:     doc.StoreFax,
		Email:   doc.StoreEmail,
		Website: doc.StoreWebsite,
		TaxID:   doc.StoreTaxID,
		LogoURL: doc.StoreLogoURL,
	})
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

// RenderWHTCert generates the WHT certificate HTML for a given document.
// receiverType: "individual" → ภ.ง.ด.3, "company" → ภ.ง.ด.53
func (s Service) RenderWHTCert(ctx context.Context, actor auth.Claims, storeID, id string, opts WHTCertOptions) (string, error) {
	doc, err := s.GetDocument(ctx, actor, storeID, id)
	if err != nil {
		return "", err
	}

	var store struct {
		Name    string
		Address string
		TaxID   string
	}
	_ = s.db.Raw(
		"SELECT name, COALESCE(address,'') AS address, COALESCE(tax_id,'') AS tax_id FROM stores WHERE id = ?",
		storeID,
	).Scan(&store)
	// Note: WHT cert uses payer name/address/TaxID only; phone/fax/email/website not shown

	formNo := "ภ.ง.ด.53"
	if opts.ReceiverType == "individual" {
		formNo = "ภ.ง.ด.3"
	}

	incomeType := opts.IncomeType
	if incomeType == "" {
		incomeType = "เงินได้ตามมาตรา 40(8) บริการทั่วไป"
	}
	whtRate := opts.WHTRate
	if whtRate <= 0 {
		whtRate = 3
	}

	gross := doc.Subtotal // WHT base = pre-VAT subtotal (Revenue Code rule)
	whtAmount := math.Round(gross*whtRate/100*100) / 100
	net := math.Round((gross-whtAmount)*100) / 100

	payeeTaxID := ""
	if doc.CustomerTaxID != nil {
		payeeTaxID = *doc.CustomerTaxID
	}

	d := dochtml.WHTCertData{
		PayerName:    store.Name,
		PayerAddress: store.Address,
		PayerTaxID:   store.TaxID,

		PayeeName:    doc.CustomerName,
		PayeeAddress: doc.CustomerAddress,
		PayeeTaxID:   payeeTaxID,

		ReceiverType: opts.ReceiverType,
		FormNo:       formNo,

		DocumentNo:  doc.DocumentNoFull,
		PaymentDate: doc.DocumentDate,

		IncomeType:  incomeType,
		IncomeDesc:  opts.IncomeDesc,
		GrossAmount: gross,
		WHTRate:     whtRate,
		WHTAmount:   whtAmount,
		NetAmount:   net,
	}
	return dochtml.RenderWHTCertHTML(d)
}

// WHTCertOptions holds query parameters for RenderWHTCert.
type WHTCertOptions struct {
	ReceiverType string  // "individual" | "company" (default "company")
	IncomeType   string  // e.g. "เงินได้ตามมาตรา 40(8) บริการ"
	IncomeDesc   string  // optional extra description
	WHTRate      float64 // percentage, e.g. 3 (default 3)
}

func (s Service) ensureAccess(actor auth.Claims, storeID string) error {
	if storeID == "" {
		return ErrInvalidInput
	}
	return nil
}

// toDocData maps a *Document to dochtml.DocData for HTML rendering.
func toDocData(doc *Document) dochtml.DocData {
	items := make([]dochtml.DocItem, len(doc.Items))
	for i, it := range doc.Items {
		items[i] = dochtml.DocItem{
			Description:   it.Description,
			Unit:          it.Unit,
			Quantity:      it.Quantity,
			UnitPrice:     it.UnitPrice,
			DiscountValue: it.DiscountValue,
			Amount:        it.Amount,
		}
	}
	var totalDiscount float64
	for _, it := range doc.Items {
		totalDiscount += it.DiscountValue
	}

	return dochtml.DocData{
		Type:            string(doc.Type),
		DocumentNo:      doc.DocumentNo,
		DocumentNoFull:  doc.DocumentNoFull,
		DocumentDate:    doc.DocumentDate,
		DueDate:         doc.DueDate,
		CustomerName:    doc.CustomerName,
		CustomerAddress: doc.CustomerAddress,
		CustomerPhone:   doc.CustomerPhone,
		CustomerTaxID:   doc.CustomerTaxID,
		StaffName:       doc.StaffName,
		Items:           items,
		Subtotal:        doc.Subtotal,
		TotalDiscount:   totalDiscount,
		VatRate:         doc.VatRate,
		VatAmount:       doc.VatAmount,
		TotalAmount:     doc.TotalAmount,
		Notes:           doc.Notes,
	}
}
