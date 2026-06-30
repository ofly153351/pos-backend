package document

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"pos-backend/internal/idgen"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/dochtml"
	"pos-backend/internal/platform/htmlpdf"

	"gorm.io/gorm"
)

var (
	ErrNotFound          = errors.New("document not found")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidInput      = errors.New("invalid input")
	ErrNoItems           = errors.New("document must have at least one item")
	ErrBadAction         = errors.New("unknown bulk action")
	ErrInvalidConversion = errors.New("conversion not allowed for this document type")
)

// allowedConversions is the document workflow matrix: which target types a given
// source type may be converted into. Arbitrary conversions that break the
// business workflow are rejected (ErrInvalidConversion).
var allowedConversions = map[DocumentType][]DocumentType{
	TypeQuotation:     {TypeInvoice},
	TypeInvoice:       {TypeReceipt, TypeTaxInvoice, TypeDeliveryOrder, TypeCreditNote},
	TypeReceipt:       {TypeTaxInvoice, TypeCreditNote},
	TypeDeliveryOrder: {TypeInvoice},
	TypeTaxInvoice:    {TypeCreditNote},
}

func canConvert(from, to DocumentType) bool {
	for _, t := range allowedConversions[from] {
		if t == to {
			return true
		}
	}
	return false
}

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
		Name        string `gorm:"column:name"`
		Address     string `gorm:"column:address"`
		Phone       string `gorm:"column:phone"`
		Fax         string `gorm:"column:fax"`
		Email       string `gorm:"column:email"`
		Website     string `gorm:"column:website"`
		TaxID       string `gorm:"column:tax_id"`
		LogoURL     string `gorm:"column:logo_url"`
		PromptPayID string `gorm:"column:promptpay_id"`
	}
	_ = s.db.Raw(
		"SELECT name, COALESCE(address,'') AS address, COALESCE(phone,'') AS phone, COALESCE(fax,'') AS fax, COALESCE(email,'') AS email, COALESCE(website,'') AS website, COALESCE(tax_id,'') AS tax_id, COALESCE(logo_url,'') AS logo_url, COALESCE(promptpay_id,'') AS promptpay_id FROM stores WHERE id = ?",
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
	doc.StorePromptPayID = store.PromptPayID

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
	if req.CustomerID != "" {
		if err := s.db.Raw(
			"SELECT full_name, COALESCE(address,'') AS address, COALESCE(phone,'') AS phone FROM customers WHERE id = ? AND store_id = ?",
			req.CustomerID, storeID,
		).Scan(&cust).Error; err != nil || cust.FullName == "" {
			return nil, fmt.Errorf("customer not found: %w", ErrInvalidInput)
		}
	} else if req.CustomerNameOverride != "" {
		// Walk-in / receipt scenario: use name/address/phone provided directly
		cust.FullName = req.CustomerNameOverride
		cust.Address = req.CustomerAddressOverride
		cust.Phone = req.CustomerPhoneOverride
	} else {
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

	var validUntil *time.Time
	if req.ValidUntil != nil && *req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", *req.ValidUntil)
		if err != nil {
			return nil, fmt.Errorf("invalid valid_until: %w", ErrInvalidInput)
		}
		validUntil = &t
	}

	var deliveryDate *time.Time
	if req.DeliveryDate != nil && *req.DeliveryDate != "" {
		t, err := time.Parse("2006-01-02", *req.DeliveryDate)
		if err != nil {
			return nil, fmt.Errorf("invalid delivery_date: %w", ErrInvalidInput)
		}
		deliveryDate = &t
	}

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
		Type:            req.Type,
		Status:          StatusPending,
		PaymentStatus:   PaymentUnpaid,
		CustomerID:      req.CustomerID,
		CustomerName:    cust.FullName,
		CustomerAddress: cust.Address,
		CustomerPhone:   cust.Phone,
		StaffID:         actor.UserID,
		StaffName:       actor.Name,
		DocumentDate:    docDate,
		DueDate:         dueDate,
		ValidUntil:      validUntil,
		DeliveryDate:    deliveryDate,
		DeliveryAddress: req.DeliveryAddress,
		DeliveryContact: req.DeliveryContact,
		DeliveryPhone:   req.DeliveryPhone,
		SalesZone:       req.SalesZone,
		SalespersonName: req.SalespersonName,
		InvoiceRefNo:     req.InvoiceRefNo,
		PORefNo:          req.PORefNo,
		SourceDocumentID: req.SourceDocumentID,
		ShippingFee:     round2(req.ShippingFee),
		CreditTermDays:  req.CreditTermDays,
		Subtotal:       subtotal,
		VatRate:        req.VatRate,
		VatAmount:      vatAmount,
		TotalAmount:    totalAmount,
		Notes:          req.Notes,
		Items:          items,
		CreatedBy:      actor.UserID,
	}

	return s.assignNumberAndInsert(doc)
}

// assignNumberAndInsert stamps doc with the next per-store/per-type document number
// and inserts it, retrying on a unique-violation: a concurrent create (or a NextSeq
// race) can hand two documents the same number, so step the sequence forward and try
// again rather than 500-ing. Shared by CreateDocument and CreateFromSale so both speak
// the same numbering scheme.
func (s Service) assignNumberAndInsert(doc *Document) (*Document, error) {
	prefix := typePrefix(doc.Type)
	now := time.Now()
	buddhistYear := now.Year() + 543

	const maxDocNoAttempts = 6
	var createErr error
	for attempt := 0; attempt < maxDocNoAttempts; attempt++ {
		seq, _ := s.repo.NextSeq(doc.StoreID, doc.Type)
		seq += int64(attempt)
		doc.DocumentNo = fmt.Sprintf("%s-%02d%02d-%04d", prefix, now.Year()%100, int(now.Month()), seq)
		doc.DocumentNoFull = fmt.Sprintf("%s/%d/%02d/%04d", prefix, buddhistYear, int(now.Month()), seq)
		createErr = s.repo.Create(doc)
		if createErr == nil {
			return doc, nil
		}
		if !isDuplicateDocNo(createErr) {
			return nil, createErr
		}
	}
	return nil, createErr
}

// isDuplicateDocNo reports whether err is a Postgres unique-violation (SQLSTATE
// 23505) — driver-agnostic string check so the create-retry loop stays decoupled
// from the concrete pg driver type.
func isDuplicateDocNo(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}

// CreateFromSale issues a persisted customer-facing document (e.g. TAX_INVOICE) from a
// completed POS sale, copying the sale's AUTHORITATIVE stored totals instead of
// recomputing them. This is the single source of truth that guarantees the document's
// subtotal / discount / VAT / grand-total exactly match the sale's receipt.
//
// Why not reuse CreateDocument: that path recomputes the subtotal from gross unit
// prices, has no field for a whole-bill discount, and always treats VAT as exclusive —
// so a sale with a bill discount and/or VAT-inclusive (or zero-VAT) pricing came out
// with the wrong subtotal, a dropped discount and a spurious VAT line. Here every money
// figure is taken verbatim from the sale row the cashier already collected against.
func (s Service) CreateFromSale(ctx context.Context, actor auth.Claims, storeID, saleID, docType string) (*Document, error) {
	if err := s.ensureAccess(actor, storeID); err != nil {
		return nil, err
	}
	dt := DocumentType(strings.ToUpper(strings.TrimSpace(docType)))
	if dt == "" {
		dt = TypeTaxInvoice
	}

	// 1. Authoritative sale row (stored totals computed by the sale repository at sale
	//    time — the same numbers the receipt prints).
	var sr struct {
		SaleNumber         string    `gorm:"column:sale_number"`
		CustomerID         string    `gorm:"column:customer_id"`
		CustomerName       string    `gorm:"column:customer_name"`
		CustomerPhone      string    `gorm:"column:customer_phone"`
		CustomerTaxID      string    `gorm:"column:customer_tax_id"`
		Note               string    `gorm:"column:note"`
		SubtotalAmount     float64   `gorm:"column:subtotal_amount"`
		DiscountAmount     float64   `gorm:"column:discount_amount"`
		BillDiscountAmount float64   `gorm:"column:bill_discount_amount"`
		VATPercent         float64   `gorm:"column:vat_percent"`
		VATAmount          float64   `gorm:"column:vat_amount"`
		TotalAmount        float64   `gorm:"column:total_amount"`
		SoldAt             time.Time `gorm:"column:sold_at"`
		CreatedAt          time.Time `gorm:"column:created_at"`
	}
	if err := s.db.Table("sales").
		Where("id = ? AND store_id = ?", saleID, storeID).
		Take(&sr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Refuse to mint a customer-facing document (tax invoice / receipt) from a loan
	// (ยืมสินค้า) sale: a loan's sale row stays status='completed' even after the goods are
	// returned, so issuing a tax invoice would recognize VAT/revenue for borrowed goods.
	var loanCount int64
	if err := s.db.Table("credit_sales").
		Where("sale_id = ? AND type = ?", saleID, "loan").
		Count(&loanCount).Error; err != nil {
		return nil, err
	}
	if loanCount > 0 {
		return nil, ErrNotFound
	}

	// 2. Sale line items (UnitPrice is gross/pre-discount; LineTotal is what the
	//    customer paid for the line; LineDiscountTotal is the per-line discount × qty).
	var rows []struct {
		ProductID         string  `gorm:"column:product_id"`
		ProductName       string  `gorm:"column:product_name"`
		UnitType          string  `gorm:"column:unit_type"`
		Quantity          float64 `gorm:"column:quantity"`
		UnitPrice         float64 `gorm:"column:unit_price"`
		LineDiscountTotal float64 `gorm:"column:line_discount_total"`
		LineTotal         float64 `gorm:"column:line_total"`
	}
	if err := s.db.Table("sale_items").
		Where("sale_id = ?", saleID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNoItems
	}

	// 3. VAT display mirrors the receipt: it is only broken out when the store charges
	//    VAT exclusively. Inclusive / no-VAT stores show no VAT line (vat_amount stays
	//    inside the price), so the tax invoice's grand total equals the receipt's.
	taxMode := "exclusive"
	var rsv struct {
		TaxMode string `gorm:"column:tax_mode"`
	}
	if err := s.db.Table("store_receipt_settings").
		Select("tax_mode").
		Where("store_id = ?", storeID).
		Take(&rsv).Error; err == nil && strings.TrimSpace(rsv.TaxMode) != "" {
		taxMode = rsv.TaxMode
	}
	round2 := func(v float64) float64 { return math.Round(v*100) / 100 }
	var vatRate, vatAmount float64
	if taxMode == "exclusive" && sr.VATPercent > 0 {
		vatRate = sr.VATPercent
		vatAmount = round2(sr.VATAmount)
	}

	items := make([]DocumentItem, 0, len(rows))
	for _, r := range rows {
		discType := ""
		if r.LineDiscountTotal > 0 {
			discType = "AMOUNT"
		}
		unit := strings.TrimSpace(r.UnitType)
		if unit == "" {
			unit = "ชิ้น"
		}
		productID := r.ProductID
		var pid *string
		if strings.TrimSpace(productID) != "" {
			pid = &productID
		}
		items = append(items, DocumentItem{
			ID:            idgen.Generate(PrefixDocumentItem),
			ProductID:     pid,
			Description:   r.ProductName,
			Unit:          unit,
			Quantity:      r.Quantity,
			UnitPrice:     round2(r.UnitPrice),
			DiscountType:  discType,
			DiscountValue: round2(r.LineDiscountTotal),
			Amount:        round2(r.LineTotal),
		})
	}

	docDate := sr.SoldAt
	if docDate.IsZero() {
		docDate = sr.CreatedAt
	}
	if docDate.IsZero() {
		docDate = time.Now()
	}

	customerName := strings.TrimSpace(sr.CustomerName)
	if customerName == "" {
		customerName = "ลูกค้าทั่วไป"
	}
	// Enrich the buyer block from the customer master when the sale is tied to one
	// (the sale snapshot has no address); fall back to the sale snapshot otherwise.
	customerAddress := ""
	if strings.TrimSpace(sr.CustomerID) != "" {
		var cust struct {
			FullName string `gorm:"column:full_name"`
			Address  string `gorm:"column:address"`
			Phone    string `gorm:"column:phone"`
		}
		if err := s.db.Raw(
			"SELECT full_name, COALESCE(address,'') AS address, COALESCE(phone,'') AS phone FROM customers WHERE id = ? AND store_id = ?",
			sr.CustomerID, storeID,
		).Scan(&cust).Error; err == nil && cust.FullName != "" {
			customerName = cust.FullName
			customerAddress = cust.Address
			if strings.TrimSpace(sr.CustomerPhone) == "" {
				sr.CustomerPhone = cust.Phone
			}
		}
	}

	var notes *string
	if strings.TrimSpace(sr.Note) != "" {
		n := sr.Note
		notes = &n
	}
	var customerTaxID *string
	if strings.TrimSpace(sr.CustomerTaxID) != "" {
		t := sr.CustomerTaxID
		customerTaxID = &t
	}

	doc := &Document{
		ID:              idgen.Generate(PrefixDocument),
		StoreID:         storeID,
		Type:            dt,
		Status:          StatusPending,
		PaymentStatus:   PaymentUnpaid,
		CustomerID:      sr.CustomerID,
		CustomerName:    customerName,
		CustomerTaxID:   customerTaxID,
		CustomerAddress: customerAddress,
		CustomerPhone:   sr.CustomerPhone,
		StaffID:         actor.UserID,
		StaffName:       actor.Name,
		DocumentDate:    docDate,
		InvoiceRefNo:    sr.SaleNumber,
		// Money — copied verbatim from the sale, NOT recomputed.
		Subtotal:     round2(sr.SubtotalAmount),
		BillDiscount: round2(sr.BillDiscountAmount),
		VatRate:      vatRate,
		VatAmount:    vatAmount,
		TotalAmount:  round2(sr.TotalAmount),
		Notes:        notes,
		Items:        items,
		CreatedBy:    actor.UserID,
	}

	return s.assignNumberAndInsert(doc)
}

func (s Service) UpdateDocumentStatus(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateStatusRequest) error {
	if _, err := s.GetDocument(ctx, actor, storeID, id); err != nil {
		return err
	}
	return s.repo.UpdateStatus(id, req.Status)
}

// UpdatePaymentStatus sets the manual payment flag (paid/unpaid/partial) on a
// document. It does NOT touch revenue/finance — purely a tracking aid.
func (s Service) UpdatePaymentStatus(ctx context.Context, actor auth.Claims, storeID, id string, req UpdatePaymentStatusRequest) error {
	if _, err := s.GetDocument(ctx, actor, storeID, id); err != nil {
		return err
	}
	switch req.PaymentStatus {
	case PaymentUnpaid, PaymentPartial, PaymentPaid:
	default:
		return ErrInvalidInput
	}
	return s.repo.SetPaymentStatus(id, req.PaymentStatus)
}

func (s Service) DeleteDocument(ctx context.Context, actor auth.Claims, storeID, id string) error {
	if _, err := s.GetDocument(ctx, actor, storeID, id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s Service) RenderDocumentPrint(ctx context.Context, actor auth.Claims, storeID, id string, copyIdx int) (string, error) {
	doc, err := s.GetDocument(ctx, actor, storeID, id)
	if err != nil {
		return "", err
	}

	docData := toDocData(doc)
	if doc.StorePromptPayID != "" {
		docData.QRPaymentURL = dochtml.BuildPromptPayQRDataURI(doc.StorePromptPayID, doc.TotalAmount)
	}

	// Fetch bank accounts for this store
	var bankRows []struct {
		BankName    string `gorm:"column:bank_name"`
		AccountNo   string `gorm:"column:account_no"`
		AccountName string `gorm:"column:account_name"`
	}
	_ = s.db.Raw(
		"SELECT bank_name, account_no, account_name FROM store_bank_accounts WHERE store_id = ? ORDER BY created_at ASC",
		storeID,
	).Scan(&bankRows)
	bankAccounts := make([]dochtml.BankAccountInfo, len(bankRows))
	for i, r := range bankRows {
		bankAccounts[i] = dochtml.BankAccountInfo{
			BankName:    r.BankName,
			AccountNo:   r.AccountNo,
			AccountName: r.AccountName,
		}
	}

	return dochtml.RenderUnifiedDocumentCopies(docData, dochtml.StoreInfo{
		Name:         doc.StoreName,
		Address:      doc.StoreAddress,
		Phone:        doc.StorePhone,
		Fax:          doc.StoreFax,
		Email:        doc.StoreEmail,
		Website:      doc.StoreWebsite,
		TaxID:        doc.StoreTaxID,
		LogoURL:      doc.StoreLogoURL,
		PromptPayID:  doc.StorePromptPayID,
		BankAccounts: bankAccounts,
	}, copyIdx)
}

// RenderDocumentPDF produces the downloadable PDF by rendering the SAME unified
// HTML as the on-screen preview / print, then converting it with headless Chrome.
// The PDF is therefore byte-for-byte the same layout as the preview (no separate
// gofpdf renderer to drift). copyIdx selects one copy or the whole set.
func (s Service) RenderDocumentPDF(ctx context.Context, actor auth.Claims, storeID, id string, copyIdx int) ([]byte, error) {
	html, err := s.RenderDocumentPrint(ctx, actor, storeID, id, copyIdx)
	if err != nil {
		return nil, err
	}
	return htmlpdf.Render(ctx, html)
}

// RelatedDocuments returns every document in the same conversion family as id —
// the lineage reachable through source_document_id links (e.g. Quotation → Invoice
// → Delivery Order → Tax Invoice). It walks up to the family root, then collects
// the whole subtree, ordered chronologically for a timeline view. Both walks are
// bounded so a malformed/cyclic link graph can never loop forever.
func (s Service) RelatedDocuments(ctx context.Context, actor auth.Claims, storeID, id string) ([]RelatedDoc, error) {
	if err := s.ensureAccess(actor, storeID); err != nil {
		return nil, err
	}

	// 1. Climb to the family root following source_document_id.
	root := id
	seen := map[string]bool{id: true}
	for i := 0; i < 50; i++ {
		var row struct{ SourceDocumentID *string }
		s.db.WithContext(ctx).Model(&Document{}).
			Select("source_document_id").
			Where("store_id = ? AND id = ?", storeID, root).
			Scan(&row)
		if row.SourceDocumentID == nil || *row.SourceDocumentID == "" || seen[*row.SourceDocumentID] {
			break
		}
		var cnt int64
		s.db.WithContext(ctx).Model(&Document{}).
			Where("store_id = ? AND id = ?", storeID, *row.SourceDocumentID).
			Count(&cnt)
		if cnt == 0 {
			break // dangling parent — stop here
		}
		root = *row.SourceDocumentID
		seen[*row.SourceDocumentID] = true
	}

	// 2. Breadth-first collect the whole subtree under the root.
	inFamily := map[string]bool{root: true}
	family := []string{root}
	frontier := []string{root}
	for len(frontier) > 0 && len(family) < 200 {
		var kids []string
		s.db.WithContext(ctx).Model(&Document{}).
			Where("store_id = ? AND source_document_id IN ?", storeID, frontier).
			Pluck("id", &kids)
		next := kids[:0:0]
		for _, k := range kids {
			if !inFamily[k] {
				inFamily[k] = true
				family = append(family, k)
				next = append(next, k)
			}
		}
		frontier = next
	}

	// 3. Project the family, ordered as a chronological lifecycle.
	var out []RelatedDoc
	s.db.WithContext(ctx).Model(&Document{}).
		Select("id, document_no, document_no_full, type, status, payment_status, document_date, total_amount, source_document_id").
		Where("store_id = ? AND id IN ?", storeID, family).
		Order("document_date ASC, created_at ASC").
		Scan(&out)
	return out, nil
}

func (s Service) BulkAction(ctx context.Context, actor auth.Claims, storeID string, req BulkActionRequest) error {
	if err := s.ensureAccess(actor, storeID); err != nil {
		return err
	}
	switch req.Action {
	case "DELETE":
		return s.repo.BulkDelete(storeID, req.IDs)
	case "SET_STATUS":
		if req.Status == nil {
			return ErrInvalidInput
		}
		return s.repo.BulkSetStatus(storeID, req.IDs, *req.Status)
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
	if actor.Role == auth.RolePlatformAdmin {
		return nil
	}
	var count int64
	if err := s.db.Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ? AND status <> 'suspended'", storeID, actor.UserID, []string{"owner", "manager", "cashier"}).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrForbidden
	}
	return nil
}

// PayInvoice marks an INVOICE as paid.
// If a DELIVERY_ORDER linked to this invoice already exists, skip TAX_INVOICE creation
// because the DO serves as the combined delivery note + tax invoice.
func (s Service) PayInvoice(ctx context.Context, actor auth.Claims, storeID, id string) (*Document, error) {
	src, err := s.GetDocument(ctx, actor, storeID, id)
	if err != nil {
		return nil, err
	}
	if src.Type != TypeInvoice {
		return nil, fmt.Errorf("document is not an invoice: %w", ErrInvalidInput)
	}
	if err := s.repo.MarkPaid(id); err != nil {
		return nil, err
	}

	// ถ้ามี DO ที่สร้างจาก invoice นี้อยู่แล้ว → ไม่สร้าง TAX_INVOICE ซ้ำ
	var doCount int64
	s.db.Model(&Document{}).
		Where("source_document_id = ? AND type = ?", id, TypeDeliveryOrder).
		Count(&doCount)
	if doCount > 0 {
		src.Status = StatusCompleted
		src.PaymentStatus = PaymentPaid
		return src, nil
	}

	taxDoc, err := s.createTaxInvoiceFrom(ctx, actor, storeID, src)
	if err != nil {
		return nil, err
	}
	if err := s.repo.MarkPaid(taxDoc.ID); err != nil {
		return nil, err
	}
	taxDoc.Status = StatusCompleted
	taxDoc.PaymentStatus = PaymentPaid
	return taxDoc, nil
}

// Convert creates a new document of targetType from an existing one, copying its
// line items, customer snapshot, VAT rate and notes, and linking back to the
// source via SourceDocumentID. The (source → target) pair must be permitted by
// allowedConversions. This is a document-level copy — pricing / VAT / accounting
// are NOT altered (a CREDIT_NOTE is copied as-is, not auto-negated).
func (s Service) Convert(ctx context.Context, actor auth.Claims, storeID, id string, target DocumentType) (*Document, error) {
	src, err := s.GetDocument(ctx, actor, storeID, id)
	if err != nil {
		return nil, err
	}
	if !canConvert(src.Type, target) {
		return nil, fmt.Errorf("cannot convert %s to %s: %w", src.Type, target, ErrInvalidConversion)
	}

	// buildConversionRequest + applyCustomerShipping resolve only the NON-money fields
	// (customer snapshot, delivery / reference fields, dates, notes, lineage link).
	req := buildConversionRequest(src, target)
	// A DELIVERY_ORDER ships to the customer's saved delivery profile, not their
	// billing snapshot — overlay it when one exists (blank fields keep the fallback).
	if target == TypeDeliveryOrder && src.CustomerID != "" {
		s.applyCustomerShipping(ctx, storeID, src.CustomerID, &req)
	}

	// Money + items are copied VERBATIM from the source — the source's stored totals are
	// already authoritative (e.g. it may itself have come from a sale via CreateFromSale).
	// Recomputing here would drop the whole-bill discount and re-derive VAT exclusively,
	// the same defect CreateFromSale fixes for the sale → document path. CREDIT_NOTE keeps
	// the positive amounts and is distinguished by type, not by negating values.
	items := make([]DocumentItem, len(src.Items))
	for i, it := range src.Items {
		items[i] = DocumentItem{
			ID:            idgen.Generate(PrefixDocumentItem),
			ProductID:     it.ProductID,
			Description:   it.Description,
			Unit:          it.Unit,
			Quantity:      it.Quantity,
			UnitPrice:     it.UnitPrice,
			DiscountType:  it.DiscountType,
			DiscountValue: it.DiscountValue,
			Amount:        it.Amount,
		}
	}

	docDate, derr := time.Parse("2006-01-02", req.DocumentDate)
	if derr != nil {
		docDate = time.Now()
	}
	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		if t, perr := time.Parse("2006-01-02", *req.DueDate); perr == nil {
			dueDate = &t
		}
	}

	doc := &Document{
		ID:              idgen.Generate(PrefixDocument),
		StoreID:         storeID,
		Type:            target,
		Status:          StatusPending,
		PaymentStatus:   PaymentUnpaid,
		CustomerID:      src.CustomerID,
		CustomerName:    src.CustomerName,
		CustomerTaxID:   src.CustomerTaxID,
		CustomerAddress: src.CustomerAddress,
		CustomerPhone:   src.CustomerPhone,
		StaffID:         actor.UserID,
		StaffName:       actor.Name,
		DocumentDate:    docDate,
		DueDate:         dueDate,
		DeliveryAddress: req.DeliveryAddress,
		DeliveryContact: req.DeliveryContact,
		DeliveryPhone:   req.DeliveryPhone,
		InvoiceRefNo:     req.InvoiceRefNo,
		SourceDocumentID: req.SourceDocumentID,
		// Money — verbatim from the source, NOT recomputed.
		Subtotal:     src.Subtotal,
		BillDiscount: src.BillDiscount,
		VatRate:      src.VatRate,
		VatAmount:    src.VatAmount,
		TotalAmount:  src.TotalAmount,
		Notes:        req.Notes,
		Items:        items,
		CreatedBy:    actor.UserID,
	}
	return s.assignNumberAndInsert(doc)
}

// applyCustomerShipping overlays the customer's shipping profile onto a delivery
// order request. Each field falls back to whatever buildConversionRequest already
// set (the billing snapshot) when the shipping value is blank.
func (s Service) applyCustomerShipping(ctx context.Context, storeID, customerID string, req *CreateDocumentRequest) {
	var sh struct {
		Contact    string
		Phone      string
		Address    string
		Province   string
		District   string
		PostalCode string
	}
	_ = s.db.WithContext(ctx).Raw(
		`SELECT COALESCE(shipping_contact,'') AS contact, COALESCE(shipping_phone,'') AS phone,
		        COALESCE(shipping_address,'') AS address, COALESCE(shipping_province,'') AS province,
		        COALESCE(shipping_district,'') AS district, COALESCE(shipping_postal_code,'') AS postal_code
		   FROM customers WHERE id = ? AND store_id = ?`,
		customerID, storeID,
	).Scan(&sh)

	if sh.Contact != "" {
		req.DeliveryContact = sh.Contact
	}
	if sh.Phone != "" {
		req.DeliveryPhone = sh.Phone
	}
	if addr := joinNonEmpty(" ", sh.Address, sh.District, sh.Province, sh.PostalCode); addr != "" {
		req.DeliveryAddress = addr
	}
}

// joinNonEmpty joins the trimmed, non-blank parts with sep.
func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, sep)
}

// buildConversionRequest maps a source document to a CreateDocumentRequest for the
// target type. SourceDocumentID is always set (invisible link powering the document
// timeline); only DELIVERY_ORDER carries the extra visible delivery / reference
// fields, preserving the previously-rendered output of the other conversions.
func buildConversionRequest(src *Document, target DocumentType) CreateDocumentRequest {
	items := make([]CreateDocumentItemInput, len(src.Items))
	for i, it := range src.Items {
		items[i] = CreateDocumentItemInput{
			ProductID:     it.ProductID,
			Description:   it.Description,
			Unit:          it.Unit,
			Quantity:      it.Quantity,
			UnitPrice:     it.UnitPrice,
			DiscountType:  it.DiscountType,
			DiscountValue: it.DiscountValue,
		}
	}
	srcID := src.ID
	req := CreateDocumentRequest{
		Type:       target,
		CustomerID: src.CustomerID,
		// Carry the customer snapshot so walk-in source docs (empty CustomerID)
		// still pass CreateDocument's customer resolution. Ignored when CustomerID set.
		CustomerNameOverride:    src.CustomerName,
		CustomerAddressOverride: src.CustomerAddress,
		CustomerPhoneOverride:   src.CustomerPhone,
		DocumentDate:            time.Now().Format("2006-01-02"),
		VatRate:                 src.VatRate,
		Notes:                   src.Notes,
		SourceDocumentID:        &srcID,
		Items:                   items,
	}
	if target == TypeDeliveryOrder {
		req.InvoiceRefNo = src.DocumentNoFull
		req.DeliveryAddress = src.CustomerAddress
		req.DeliveryContact = src.CustomerName
		req.DeliveryPhone = src.CustomerPhone
		if src.DueDate != nil {
			d := src.DueDate.Format("2006-01-02")
			req.DueDate = &d
		}
	}
	return req
}

func (s Service) createTaxInvoiceFrom(ctx context.Context, actor auth.Claims, storeID string, src *Document) (*Document, error) {
	// Delegate to Convert so the tax invoice inherits the invoice's totals verbatim
	// (bill discount + VAT treatment preserved) instead of being recomputed.
	return s.Convert(ctx, actor, storeID, src.ID, TypeTaxInvoice)
}

// ConvertToTaxInvoice creates a TAX_INVOICE from an existing INVOICE (without marking paid).
func (s Service) ConvertToTaxInvoice(ctx context.Context, actor auth.Claims, storeID, id string) (*Document, error) {
	return s.Convert(ctx, actor, storeID, id, TypeTaxInvoice)
}

// ConvertToDeliveryOrder creates a DELIVERY_ORDER from an existing INVOICE.
func (s Service) ConvertToDeliveryOrder(ctx context.Context, actor auth.Claims, storeID, id string) (*Document, error) {
	return s.Convert(ctx, actor, storeID, id, TypeDeliveryOrder)
}

// ConvertQuotation creates an INVOICE document from an existing QUOTATION.
func (s Service) ConvertQuotation(ctx context.Context, actor auth.Claims, storeID, id string) (*Document, error) {
	return s.Convert(ctx, actor, storeID, id, TypeInvoice)
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
	// A whole-bill discount (set when the document was issued from a sale) is shown
	// combined with the per-item discounts on the single "ส่วนลด" summary line, so the
	// document matches the originating receipt.
	totalDiscount += doc.BillDiscount

	preVat := math.Round((doc.Subtotal-totalDiscount)*100) / 100

	return dochtml.DocData{
		Type:            string(doc.Type),
		DocumentNo:      doc.DocumentNo,
		DocumentNoFull:  doc.DocumentNoFull,
		DocumentDate:    doc.DocumentDate,
		DueDate:         doc.DueDate,
		ValidUntil:      doc.ValidUntil,
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
		PreVatAmount:    preVat,
		Notes:           doc.Notes,
		// Delivery order fields
		DeliveryDate:    doc.DeliveryDate,
		DeliveryAddress: doc.DeliveryAddress,
		DeliveryContact: doc.DeliveryContact,
		DeliveryPhone:   doc.DeliveryPhone,
		SalesZone:       doc.SalesZone,
		SalespersonName: doc.SalespersonName,
		InvoiceRefNo:    doc.InvoiceRefNo,
		PORefNo:         doc.PORefNo,
		ShippingFee:     doc.ShippingFee,
		CreditTermDays:  doc.CreditTermDays,
	}
}
