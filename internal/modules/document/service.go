package document

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
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
	ErrAlreadyConverted  = errors.New("document has already been converted to this type")
)

// fieldValidationError carries field-level failures so the handler can answer
// HTTP 422 with indexed fields (e.g. items[2].unit_price) instead of a generic
// 400. It is the single validation failure carrier for CreateDocument.
type fieldValidationError struct {
	fields []validationField
}

type validationField struct {
	field   string
	message string
}

func (e *fieldValidationError) Error() string { return "invalid document line(s)" }

func newFieldValidation(fields ...validationField) error {
	return &fieldValidationError{fields: fields}
}

// allowedConversions is the document workflow matrix: which target types a given
// source type may be converted into. Arbitrary conversions that break the
// business workflow are rejected (ErrInvalidConversion).
var allowedConversions = map[DocumentType][]DocumentType{
	TypeQuotation:     {TypeInvoice},
	TypeInvoice:       {TypeReceipt, TypeTaxInvoice, TypeDeliveryOrder, TypeCreditNote},
	TypeReceipt:       {TypeTaxInvoice, TypeCreditNote},
	TypeDeliveryOrder: {TypeInvoice, TypeReceipt},
	TypeBill:          {TypeReceipt},
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

// validateCreateLines enforces the document line business rules server-side
// (H-03): every line needs a product OR an approved description, a positive
// quantity, a non-negative unit price, a coherent discount and a supported VAT
// rate. Failures return a fieldValidationError with indexed field names
// (items[i].unit_price) so the handler can answer 422.
func validateCreateLines(req CreateDocumentRequest) error {
	var fields []validationField
	for i, inp := range req.Items {
		desc := strings.TrimSpace(inp.Description)
		hasProduct := inp.ProductID != nil && strings.TrimSpace(*inp.ProductID) != ""
		// A valid line is a real product or a free-text description line. A row
		// with neither (stale draft / leftover) is rejected. quantity and price
		// must be valid on both kinds.
		if !hasProduct && desc == "" {
			fields = append(fields, validationField{field: fmt.Sprintf("items[%d].description", i), message: "description or product is required"})
		}
		if inp.Quantity <= 0 {
			fields = append(fields, validationField{field: fmt.Sprintf("items[%d].quantity", i), message: "quantity must be greater than 0"})
		}
		if inp.UnitPrice < 0 {
			fields = append(fields, validationField{field: fmt.Sprintf("items[%d].unit_price", i), message: "unit price cannot be negative"})
		}
		switch inp.DiscountType {
		case "", "PERCENT", "AMOUNT":
		default:
			fields = append(fields, validationField{field: fmt.Sprintf("items[%d].discount_type", i), message: "discount type must be PERCENT or AMOUNT"})
		}
		if inp.DiscountType == "PERCENT" && (inp.DiscountValue < 0 || inp.DiscountValue > 100) {
			fields = append(fields, validationField{field: fmt.Sprintf("items[%d].discount_value", i), message: "percent discount must be between 0 and 100"})
		}
		if inp.DiscountType == "AMOUNT" && inp.DiscountValue < 0 {
			fields = append(fields, validationField{field: fmt.Sprintf("items[%d].discount_value", i), message: "discount cannot be negative"})
		}
	}
	if req.VatRate < 0 || req.VatRate > 100 {
		fields = append(fields, validationField{field: "vat_rate", message: "vat_rate must be between 0 and 100"})
	}
	if len(fields) > 0 {
		return newFieldValidation(fields...)
	}
	return nil
}

func (s Service) CreateDocument(ctx context.Context, actor auth.Claims, storeID string, req CreateDocumentRequest) (*Document, error) {
	if len(req.Items) == 0 {
		return nil, ErrNoItems
	}

	// H-03 server-side per-line validation. Frontend guards are UX only — the
	// server is the authority: reject empty/stale draft rows, non-positive
	// quantity, negative price, malformed discounts and unsupported VAT.
	if err := validateCreateLines(req); err != nil {
		return nil, err
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
	if req.PriceValidityDays != nil && *req.PriceValidityDays < 0 {
		return nil, fmt.Errorf("price_validity_days must be >= 0: %w", ErrInvalidInput)
	}
	if req.DeliveryLeadTimeDays != nil && *req.DeliveryLeadTimeDays < 0 {
		return nil, fmt.Errorf("delivery_lead_time_days must be >= 0: %w", ErrInvalidInput)
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
	if req.PriceValidityDays != nil {
		t := docDate.AddDate(0, 0, *req.PriceValidityDays)
		validUntil = &t
	} else if req.ValidUntil != nil && *req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", *req.ValidUntil)
		if err != nil {
			return nil, fmt.Errorf("invalid valid_until: %w", ErrInvalidInput)
		}
		validUntil = &t
	}

	var poReceivedDate, expectedDeliveryDate *time.Time
	if req.POReceivedDate != nil && *req.POReceivedDate != "" {
		t, parseErr := time.Parse("2006-01-02", *req.POReceivedDate)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid po_received_date: %w", ErrInvalidInput)
		}
		poReceivedDate = &t
	}

	calculatedValidUntil, calculatedExpected := calculateTermDates(docDate, req.PriceValidityDays, req.DeliveryLeadTimeDays, poReceivedDate)
	if calculatedValidUntil != nil {
		validUntil = calculatedValidUntil
	}
	expectedDeliveryDate = calculatedExpected

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
		ID:                   idgen.Generate(PrefixDocument),
		StoreID:              storeID,
		Type:                 req.Type,
		Status:               StatusPending,
		PaymentStatus:        PaymentUnpaid,
		CustomerID:           req.CustomerID,
		CustomerName:         cust.FullName,
		CustomerAddress:      cust.Address,
		CustomerPhone:        cust.Phone,
		StaffID:              actor.UserID,
		StaffName:            actor.Name,
		DocumentDate:         docDate,
		DueDate:              dueDate,
		ValidUntil:           validUntil,
		PriceValidityDays:    req.PriceValidityDays,
		DeliveryDate:         deliveryDate,
		DeliveryLeadTimeDays: req.DeliveryLeadTimeDays,
		POReceivedDate:       poReceivedDate,
		ExpectedDeliveryDate: expectedDeliveryDate,
		DeliveryAddress:      req.DeliveryAddress,
		DeliveryContact:      req.DeliveryContact,
		DeliveryPhone:        req.DeliveryPhone,
		SalesZone:            req.SalesZone,
		SalespersonName:      req.SalespersonName,
		InvoiceRefNo:         req.InvoiceRefNo,
		PORefNo:              req.PORefNo,
		SourceDocumentID:     req.SourceDocumentID,
		ShippingFee:          round2(req.ShippingFee),
		CreditTermDays:       req.CreditTermDays,
		Subtotal:             subtotal,
		VatRate:              req.VatRate,
		VatAmount:            vatAmount,
		TotalAmount:          totalAmount,
		Notes:                req.Notes,
		Items:                items,
		CreatedBy:            actor.UserID,
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
		doc.DocumentNo = fmt.Sprintf("%s%d%02d-%04d", prefix, buddhistYear, int(now.Month()), seq)
		doc.DocumentNoFull = fmt.Sprintf("%s%d%02d-%04d", prefix, buddhistYear, int(now.Month()), seq)
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

func (s Service) SetReceiptTemplate(ctx context.Context, actor auth.Claims, storeID, id string, template int) error {
	if template != 1 && template != 2 {
		return fmt.Errorf("receipt template must be 1 or 2")
	}
	if _, err := s.GetDocument(ctx, actor, storeID, id); err != nil {
		return err
	}
	return s.repo.SetReceiptTemplate(id, template)
}

func (s Service) RenderDocumentPrint(ctx context.Context, actor auth.Claims, storeID, id string, copyIdx int, receiptTemplate int) (string, error) {
	doc, err := s.GetDocument(ctx, actor, storeID, id)
	if err != nil {
		return "", err
	}
	if receiptTemplate == 0 {
		receiptTemplate = doc.ReceiptTemplate
		if receiptTemplate == 0 {
			receiptTemplate = 1
		}
	}

	docData := toDocData(doc)
	if doc.Type == TypeReceipt && receiptTemplate == 2 {
		docData.ReceiptTemplate = 2
		docData.PaymentDate = doc.DocumentDate
		docData.PaymentAmount = doc.TotalAmount
		billingRef, deliveryRef := s.resolveReceiptReferences(doc)
		docData.PaymentDescription = buildReceiptPaymentDescription(
			billingRef,
			deliveryRef,
			doc.DocumentNoFull,
		)
		docData.DeliveryRefNo = deliveryRef
		docData.BillingRefNo = billingRef
		settlements, settlementErr := s.repo.ListReceiptSettlements(doc.ID)
		if settlementErr != nil {
			return "", settlementErr
		}
		docData.ReceiptSettlements = make([]dochtml.ReceiptSettlementRow, 0, len(settlements))
		for _, settlement := range settlements {
			row := dochtml.ReceiptSettlementRow{Amount: settlement.AppliedAmount}
			if settlement.BillingDocument != nil {
				row.BillingRef = settlement.BillingDocument.DocumentNoFull
			}
			if settlement.DeliveryOrder != nil {
				row.DeliveryRef = settlement.DeliveryOrder.DocumentNoFull
			}
			docData.ReceiptSettlements = append(docData.ReceiptSettlements, row)
		}
	}
	// ใบแจ้งหนี้ที่สร้างจากใบเสนอราคา → แถว "อ้างอิงใบเสนอราคา" ในหัวเอกสาร
	docData.QuotationRefNo = s.resolveQuotationRef(doc)

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

	storeInfo := dochtml.StoreInfo{
		Name:         doc.StoreName,
		Address:      doc.StoreAddress,
		Phone:        doc.StorePhone,
		Fax:          doc.StoreFax,
		Email:        doc.StoreEmail,
		Website:      doc.StoreWebsite,
		TaxID:        doc.StoreTaxID,
		LogoURL:      inlineImageDataURI(ctx, doc.StoreLogoURL),
		PromptPayID:  doc.StorePromptPayID,
		BankAccounts: bankAccounts,
	}
	if doc.Type == TypeReceipt && receiptTemplate == 2 {
		// Type 2 prints one payment row per persisted settlement. Legacy receipts
		// without settlement rows retain the single-row fallback.
		if len(docData.ReceiptSettlements) > 0 {
			docData.Items = make([]dochtml.DocItem, 0, len(docData.ReceiptSettlements))
			for _, settlement := range docData.ReceiptSettlements {
				description := buildReceiptPaymentDescription(
					settlement.BillingRef,
					settlement.DeliveryRef,
					doc.DocumentNoFull,
				)
				docData.Items = append(docData.Items, dochtml.DocItem{
					Description: description,
					Quantity:    1,
					Unit:        "รายการ",
					Amount:      settlement.Amount,
					UnitPrice:   settlement.Amount,
				})
			}
		} else {
			docData.Items = []dochtml.DocItem{{
				Description: docData.PaymentDescription,
				Quantity:    1,
				Unit:        "รายการ",
				Amount:      docData.PaymentAmount,
				UnitPrice:   docData.PaymentAmount,
			}}
		}
		docData.Subtotal = docData.PaymentAmount
		docData.VatAmount = 0
		docData.TotalAmount = docData.PaymentAmount
	}
	return dochtml.RenderUnifiedDocumentCopies(docData, storeInfo, copyIdx)
}

// inlineImageDataURI fetches an image URL and returns it as a base64 data: URI so
// it embeds directly in the HTML. The store logo lives on an internal asset host
// (MinIO, e.g. http://127.0.0.1:9000) that headless Chrome can't reliably reach
// when generating the PDF — embedding it guarantees the logo shows in both the
// preview and the downloaded PDF. On any failure it returns the URL unchanged.
func inlineImageDataURI(ctx context.Context, rawURL string) string {
	if rawURL == "" || strings.HasPrefix(rawURL, "data:") {
		return rawURL
	}
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return rawURL
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rawURL
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return rawURL
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5MB safety cap
	if err != nil || len(data) == 0 {
		return rawURL
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// RenderDocumentPDF produces the downloadable PDF by rendering the SAME unified
// HTML as the on-screen preview / print, then converting it with headless Chrome.
// The PDF is therefore byte-for-byte the same layout as the preview (no separate
// gofpdf renderer to drift). copyIdx selects one copy or the whole set.
func (s Service) RenderDocumentPDF(ctx context.Context, actor auth.Claims, storeID, id string, copyIdx int, receiptTemplate int) ([]byte, error) {
	html, err := s.RenderDocumentPrint(ctx, actor, storeID, id, copyIdx, receiptTemplate)
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
func (s Service) Convert(ctx context.Context, actor auth.Claims, storeID, id string, target DocumentType, deliveryDateOverride ...*string) (*Document, error) {
	src, err := s.GetDocument(ctx, actor, storeID, id)
	if err != nil {
		return nil, err
	}
	if !canConvert(src.Type, target) {
		return nil, fmt.Errorf("cannot convert %s to %s: %w", src.Type, target, ErrInvalidConversion)
	}
	if existing, err := s.repo.FindBySourceAndType(src.ID, target); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyConverted, existing.DocumentNoFull)
	}

	// buildConversionRequest + applyCustomerShipping resolve only the NON-money fields
	// (customer snapshot, delivery / reference fields, dates, notes, lineage link).
	req := buildConversionRequest(src, target, deliveryDateOverride...)
	// For DELIVERY_ORDER the optional second override is the purchase-order
	// reference entered by the user during the invoice → DO conversion modal.
	if target == TypeDeliveryOrder && len(deliveryDateOverride) > 1 && deliveryDateOverride[1] != nil {
		req.PORefNo = strings.TrimSpace(*deliveryDateOverride[1])
	}
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
	var deliveryDate *time.Time
	if req.DeliveryDate != nil && *req.DeliveryDate != "" {
		if t, perr := time.Parse("2006-01-02", *req.DeliveryDate); perr == nil {
			deliveryDate = &t
		}
	}

	doc := &Document{
		ID:                   idgen.Generate(PrefixDocument),
		StoreID:              storeID,
		Type:                 target,
		Status:               StatusPending,
		PaymentStatus:        PaymentUnpaid,
		CustomerID:           src.CustomerID,
		CustomerName:         src.CustomerName,
		CustomerTaxID:        src.CustomerTaxID,
		CustomerAddress:      src.CustomerAddress,
		CustomerPhone:        src.CustomerPhone,
		StaffID:              actor.UserID,
		StaffName:            actor.Name,
		DocumentDate:         docDate,
		DueDate:              dueDate,
		DeliveryDate:         deliveryDate,
		ValidUntil:           src.ValidUntil,
		PriceValidityDays:    src.PriceValidityDays,
		DeliveryLeadTimeDays: src.DeliveryLeadTimeDays,
		POReceivedDate:       src.POReceivedDate,
		ExpectedDeliveryDate: src.ExpectedDeliveryDate,
		DeliveryAddress:      req.DeliveryAddress,
		DeliveryContact:      req.DeliveryContact,
		DeliveryPhone:        req.DeliveryPhone,
		InvoiceRefNo:         req.InvoiceRefNo,
		PORefNo:              req.PORefNo,
		SourceDocumentID:     req.SourceDocumentID,
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

// ConvertReceiptFromBills creates one receipt for multiple billing notices. Each
// selected BILL contributes its delivery-order rows to receipt_settlements, so Type 2
// can render one payment row per BN/DO relationship without parsing display text.
func (s Service) ConvertReceiptFromBills(ctx context.Context, actor auth.Claims, storeID string, billIDs []string, receiptTemplate int) (*Document, error) {
	if len(billIDs) == 0 {
		return nil, fmt.Errorf("at least one billing document is required: %w", ErrInvalidInput)
	}
	if receiptTemplate != 1 && receiptTemplate != 2 {
		return nil, fmt.Errorf("receipt template must be 1 or 2: %w", ErrInvalidInput)
	}

	seen := make(map[string]bool, len(billIDs))
	bills := make([]*Document, 0, len(billIDs))
	for _, id := range billIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		bill, err := s.GetDocument(ctx, actor, storeID, id)
		if err != nil {
			return nil, err
		}
		if bill.Type != TypeBill {
			return nil, fmt.Errorf("document %s is not a billing notice: %w", id, ErrInvalidInput)
		}
		if bill.Status == StatusCancelled || bill.PaymentStatus == PaymentPaid || bill.PaymentStatus == PaymentPartial {
			return nil, fmt.Errorf("billing document %s is not fully payable: %w", bill.DocumentNoFull, ErrInvalidInput)
		}
		if existing, err := s.repo.FindBySourceAndType(bill.ID, TypeReceipt); err != nil {
			return nil, err
		} else if existing != nil {
			return nil, fmt.Errorf("%w: %s", ErrAlreadyConverted, existing.DocumentNoFull)
		}
		if len(bills) > 0 && bill.CustomerID != bills[0].CustomerID {
			return nil, fmt.Errorf("billing documents must belong to the same customer: %w", ErrInvalidInput)
		}
		bills = append(bills, bill)
	}

	primary := bills[0]
	var subtotal, billDiscount, vatAmount, totalAmount float64
	items := make([]DocumentItem, 0)
	for _, bill := range bills {
		subtotal += bill.Subtotal
		billDiscount += bill.BillDiscount
		vatAmount += bill.VatAmount
		totalAmount += bill.TotalAmount
		for _, item := range bill.Items {
			items = append(items, DocumentItem{
				ID:            idgen.Generate(PrefixDocumentItem),
				ProductID:     item.ProductID,
				Description:   item.Description,
				Unit:          item.Unit,
				Quantity:      item.Quantity,
				UnitPrice:     item.UnitPrice,
				DiscountType:  item.DiscountType,
				DiscountValue: item.DiscountValue,
				Amount:        item.Amount,
			})
		}
	}

	receipt := &Document{
		ID:               idgen.Generate(PrefixDocument),
		StoreID:          storeID,
		Type:             TypeReceipt,
		Status:           StatusPending,
		PaymentStatus:    PaymentUnpaid,
		ReceiptTemplate:  receiptTemplate,
		CustomerID:       primary.CustomerID,
		CustomerName:     primary.CustomerName,
		CustomerTaxID:    primary.CustomerTaxID,
		CustomerAddress:  primary.CustomerAddress,
		CustomerPhone:    primary.CustomerPhone,
		StaffID:          actor.UserID,
		StaffName:        actor.Name,
		DocumentDate:     time.Now(),
		InvoiceRefNo:     primary.DocumentNoFull,
		SourceDocumentID: &primary.ID,
		Subtotal:         subtotal,
		BillDiscount:     billDiscount,
		VatRate:          primary.VatRate,
		VatAmount:        vatAmount,
		TotalAmount:      totalAmount,
		Notes:            primary.Notes,
		Items:            items,
		CreatedBy:        actor.UserID,
	}

	settlements := make([]ReceiptSettlement, 0)
	for _, bill := range bills {
		addedDelivery := false
		for _, item := range bill.Items {
			var delivery Document
			err := s.db.WithContext(ctx).
				Where("store_id = ? AND type = ? AND document_no_full = ?", storeID, TypeDeliveryOrder, strings.TrimSpace(item.Description)).
				First(&delivery).Error
			if err != nil {
				continue
			}
			billingID := bill.ID
			deliveryID := delivery.ID
			settlements = append(settlements, ReceiptSettlement{
				ID:                idgen.Generate(PrefixDocumentItem),
				ReceiptDocumentID: receipt.ID,
				BillingDocumentID: &billingID,
				DeliveryOrderID:   &deliveryID,
				AppliedAmount:     item.Amount,
				SortOrder:         len(settlements) + 1,
			})
			addedDelivery = true
		}
		if !addedDelivery {
			billingID := bill.ID
			settlements = append(settlements, ReceiptSettlement{
				ID:                idgen.Generate(PrefixDocumentItem),
				ReceiptDocumentID: receipt.ID,
				BillingDocumentID: &billingID,
				AppliedAmount:     bill.TotalAmount,
				SortOrder:         len(settlements) + 1,
			})
		}
	}

	receiptItems := receipt.Items
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		prefix := fmt.Sprintf("RCT%d%02d", now.Year()+543, now.Month())
		var seq int64
		if err := tx.Model(&Document{}).
			Where("store_id = ? AND type = ? AND document_no LIKE ?", storeID, TypeReceipt, prefix+"-%").
			Select("COALESCE(MAX(CAST(SUBSTRING(document_no FROM '[0-9]+$') AS INTEGER)), 0)").
			Scan(&seq).Error; err != nil {
			return err
		}
		receipt.DocumentNo = fmt.Sprintf("RCT%d%02d-%04d", now.Year()+543, now.Month(), seq+1)
		receipt.DocumentNoFull = receipt.DocumentNo
		receipt.Items = nil
		if err := tx.Create(receipt).Error; err != nil {
			return err
		}
		for i := range receiptItems {
			receiptItems[i].DocumentID = receipt.ID
		}
		if len(receiptItems) > 0 {
			if err := tx.Create(&receiptItems).Error; err != nil {
				return err
			}
		}
		if len(settlements) > 0 {
			if err := tx.Create(&settlements).Error; err != nil {
				return err
			}
		}
		for _, bill := range bills {
			if err := tx.Model(&Document{}).Where("id = ?", bill.ID).Updates(map[string]any{
				"payment_status": PaymentPaid,
				"updated_at":     time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	receipt.Items = receiptItems
	return receipt, nil
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
func buildConversionRequest(src *Document, target DocumentType, deliveryDate ...*string) CreateDocumentRequest {
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
	req.PriceValidityDays = src.PriceValidityDays
	req.DeliveryLeadTimeDays = src.DeliveryLeadTimeDays
	if src.POReceivedDate != nil {
		d := src.POReceivedDate.Format("2006-01-02")
		req.POReceivedDate = &d
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
		if len(deliveryDate) > 0 {
			req.DeliveryDate = deliveryDate[0]
		}
	}
	if target == TypeReceipt {
		req.InvoiceRefNo = src.DocumentNoFull
	}
	return req
}

// resolveReceiptReferences walks the source-document chain so Receipt Type 2 can show
// the billing notice and delivery order that the payment settles. A receipt created
// from a delivery order commonly has the DO as its direct source and the billing
// reference one level further up, so looking only at the direct source is insufficient.
func (s Service) resolveReceiptReferences(doc *Document) (billingRef, deliveryRef string) {
	current := doc
	seen := map[string]bool{}
	for depth := 0; current != nil && current.SourceDocumentID != nil && depth < 8; depth++ {
		sourceID := strings.TrimSpace(*current.SourceDocumentID)
		if sourceID == "" || seen[sourceID] {
			break
		}
		seen[sourceID] = true

		source, err := s.repo.FindByID(sourceID)
		if err != nil {
			log.Printf("[document] receipt reference lookup failed: receipt=%s source=%s err=%v", doc.ID, sourceID, err)
			break
		}
		switch source.Type {
		case TypeBill:
			if billingRef == "" {
				billingRef = strings.TrimSpace(source.DocumentNoFull)
			}
			// BILL rows are the selected delivery-order register. Keep the
			// first non-empty reference for the receipt description.
			if deliveryRef == "" {
				for _, item := range source.Items {
					if ref := strings.TrimSpace(item.Description); ref != "" {
						deliveryRef = ref
						break
					}
				}
			}
		case TypeDeliveryOrder:
			if deliveryRef == "" {
				deliveryRef = strings.TrimSpace(source.DocumentNoFull)
			}
			if billingRef == "" {
				ref := strings.TrimSpace(source.InvoiceRefNo)
				if ref != "" && ref != deliveryRef {
					billingRef = ref
				}
			}
		}
		current = source
	}
	return billingRef, deliveryRef
}

func buildReceiptPaymentDescription(billingRef, deliveryRef, receiptRef string) string {
	switch {
	case billingRef != "" && deliveryRef != "":
		return fmt.Sprintf("ชำระค่าสินค้าตามใบวางบิล เลขที่ %s (ใบส่งสินค้า %s)", billingRef, deliveryRef)
	case billingRef != "":
		return fmt.Sprintf("ชำระค่าสินค้าตามใบวางบิล เลขที่ %s", billingRef)
	case deliveryRef != "":
		return fmt.Sprintf("ชำระค่าสินค้าตามใบส่งสินค้า %s", deliveryRef)
	default:
		return "ชำระเงินตามเอกสาร " + receiptRef
	}
}

// quotationRefNoOf returns the visible "อ้างอิงใบเสนอราคา / Ref. Quotation" reference a
// source document contributes: only a QUOTATION does, every other source type renders
// nothing. Pure, so it is unit-testable without a DB.
func quotationRefNoOf(src *Document) string {
	if src == nil || src.Type != TypeQuotation {
		return ""
	}
	if full := strings.TrimSpace(src.DocumentNoFull); full != "" {
		return full
	}
	return strings.TrimSpace(src.DocumentNo)
}

// resolveQuotationRef loads the document `doc` was created from (SourceDocumentID) and
// returns its number when that source is a QUOTATION — the "อ้างอิงใบเสนอราคา
// (Ref. Quotation)" row on an invoice created from a quotation. Resolved at render time
// (never stored) so invoices created before this row existed display it too.
// A lookup failure degrades to "no row" but is logged, not swallowed.
func (s Service) resolveQuotationRef(doc *Document) string {
	if doc == nil || doc.SourceDocumentID == nil || strings.TrimSpace(*doc.SourceDocumentID) == "" {
		return ""
	}
	src, err := s.repo.FindByID(*doc.SourceDocumentID)
	if err != nil {
		log.Printf("[document] quotation ref lookup failed: doc=%s source=%s err=%v", doc.ID, *doc.SourceDocumentID, err)
		return ""
	}
	return quotationRefNoOf(src)
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
func (s Service) ConvertToDeliveryOrder(ctx context.Context, actor auth.Claims, storeID, id string, deliveryDate, poRefNo *string) (*Document, error) {
	return s.Convert(ctx, actor, storeID, id, TypeDeliveryOrder, deliveryDate, poRefNo)
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

	var billRows []dochtml.BillRow
	if doc.Type == TypeBill {
		billRows = make([]dochtml.BillRow, 0, len(doc.Items))
		for _, item := range doc.Items {
			if strings.TrimSpace(item.Description) == "" {
				continue
			}
			billRows = append(billRows, dochtml.BillRow{
				DocumentNo: item.Description,
				IssueDate:  doc.DocumentDate,
				DueDate:    doc.DueDate,
				Amount:     item.Amount,
			})
		}
	}

	return dochtml.DocData{
		Type:              string(doc.Type),
		DocumentNo:        doc.DocumentNo,
		DocumentNoFull:    doc.DocumentNoFull,
		DocumentDate:      doc.DocumentDate,
		DueDate:           doc.DueDate,
		ValidUntil:        doc.ValidUntil,
		PriceValidityDays: doc.PriceValidityDays,
		CustomerName:      doc.CustomerName,
		CustomerAddress:   doc.CustomerAddress,
		CustomerPhone:     doc.CustomerPhone,
		CustomerTaxID:     doc.CustomerTaxID,
		StaffName:         doc.StaffName,
		Items:             items,
		BillRows:          billRows,
		Subtotal:          doc.Subtotal,
		TotalDiscount:     totalDiscount,
		VatRate:           doc.VatRate,
		VatAmount:         doc.VatAmount,
		TotalAmount:       doc.TotalAmount,
		PreVatAmount:      preVat,
		Notes:             doc.Notes,
		// Delivery order fields
		DeliveryDate:         doc.DeliveryDate,
		DeliveryLeadTimeDays: doc.DeliveryLeadTimeDays,
		POReceivedDate:       doc.POReceivedDate,
		ExpectedDeliveryDate: doc.ExpectedDeliveryDate,
		DeliveryAddress:      doc.DeliveryAddress,
		DeliveryContact:      doc.DeliveryContact,
		DeliveryPhone:        doc.DeliveryPhone,
		SalesZone:            doc.SalesZone,
		SalespersonName:      doc.SalespersonName,
		InvoiceRefNo:         doc.InvoiceRefNo,
		PORefNo:              doc.PORefNo,
		ShippingFee:          doc.ShippingFee,
		CreditTermDays:       doc.CreditTermDays,
	}
}
