package document

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) ListDocuments(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	q := ListQuery{
		StoreID:       storeID,
		Search:        c.Query("search"),
		Type:          c.Query("type"),
		Status:        c.Query("status"),
		PaymentStatus: c.Query("payment_status"),
		CustomerID:    c.Query("customer_id"),
		StaffID:       c.Query("staff_id"),
		DateFrom:      c.Query("date_from"),
		DateTo:        c.Query("date_to"),
		Page:          c.QueryInt("page", 1),
		Limit:         c.QueryInt("limit", 20),
	}
	result, err := h.service.ListDocuments(c.UserContext(), middleware.ClaimsFromContext(c), q)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "documents fetched", result)
}

func (h Handler) GetDocument(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	doc, err := h.service.GetDocument(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "document fetched", doc)
}

func (h Handler) CreateDocument(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	doc, err := h.service.CreateDocument(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "document created", doc)
}

// CreateFromSale issues a persisted document (TAX_INVOICE by default) from a POS sale,
// mirroring the sale's authoritative totals so the document matches the receipt.
// Body: { "type": "TAX_INVOICE" }. The sale id comes from the route.
func (h Handler) CreateFromSale(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	saleID := c.Params("saleID")
	var req struct {
		Type string `json:"type"`
	}
	// Body is optional — default to TAX_INVOICE when absent/empty.
	_ = c.BodyParser(&req)
	doc, err := h.service.CreateFromSale(c.UserContext(), middleware.ClaimsFromContext(c), storeID, saleID, req.Type)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "document created from sale", doc)
}

func (h Handler) UpdateDocumentStatus(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	var req UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.service.UpdateDocumentStatus(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, req); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "status updated", nil)
}

func (h Handler) UpdateDocumentPaymentStatus(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	var req UpdatePaymentStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.service.UpdatePaymentStatus(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, req); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "payment status updated", nil)
}

func (h Handler) DeleteDocument(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	if err := h.service.DeleteDocument(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "document deleted", nil)
}

func (h Handler) PrintDocument(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	// ?copy=N selects a single copy (0-based); absent/-1 prints the whole set.
	html, err := h.service.RenderDocumentPrint(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, c.QueryInt("copy", -1))
	if err != nil {
		return writeError(c, err)
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-store")
	return c.SendString(html)
}

// GetDocumentPDF renders the document to PDF via headless Chrome from the SAME
// unified HTML as the preview/print, so the download matches exactly.
// ?copy=N selects a single copy (0-based); absent/-1 returns the whole copy set.
func (h Handler) GetDocumentPDF(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	data, err := h.service.RenderDocumentPDF(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, c.QueryInt("copy", -1))
	if err != nil {
		return writeError(c, err)
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "inline; filename=\"document.pdf\"")
	c.Set("Cache-Control", "no-store")
	return c.Send(data)
}

// GetStatementPDF generates and streams a Statement PDF for a customer + period.
// Query params: customer_id (required), period_start, period_end (YYYY-MM-DD),
//
//	bank_name, account_number, note
func (h Handler) GetStatementPDF(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	customerID := c.Query("customer_id")
	if customerID == "" {
		return httpx.Error(c, fiber.StatusBadRequest, "customer_id required", nil)
	}

	parseDateQ := func(key string, def time.Time) time.Time {
		v := c.Query(key)
		if v == "" {
			return def
		}
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return def
		}
		return t
	}
	now := time.Now()
	periodStart := parseDateQ("period_start", time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()))
	periodEnd := parseDateQ("period_end", now)

	opts := StatementPDFOptions{
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		BankName:      c.Query("bank_name"),
		AccountNumber: c.Query("account_number"),
		Note:          c.Query("note"),
	}
	data, _, err := h.service.GenerateStatementPDF(c.UserContext(), middleware.ClaimsFromContext(c), storeID, customerID, opts)
	if err != nil {
		return writeError(c, err)
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "inline; filename=\"statement.pdf\"")
	c.Set("Cache-Control", "no-store")
	return c.Send(data)
}

// PrintWHTCert generates and returns the WHT certificate HTML (ภ.ง.ด.3/53).
// Query params:
//
//	receiver_type: "individual" | "company" (default "company")
//	income_type:   free-text description of income type (default "เงินได้ตามมาตรา 40(8) บริการทั่วไป")
//	income_desc:   optional additional description
//	wht_rate:      WHT rate % (default 3)
func (h Handler) PrintWHTCert(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	opts := WHTCertOptions{
		ReceiverType: c.Query("receiver_type", "company"),
		IncomeType:   c.Query("income_type"),
		IncomeDesc:   c.Query("income_desc"),
		WHTRate:      float64(c.QueryInt("wht_rate", 3)),
	}
	html, err := h.service.RenderWHTCert(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, opts)
	if err != nil {
		return writeError(c, err)
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-store")
	return c.SendString(html)
}

func (h Handler) PayInvoice(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	taxDoc, err := h.service.PayInvoice(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "invoice paid and tax invoice created", taxDoc)
}

func (h Handler) ConvertToTaxInvoice(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	taxDoc, err := h.service.ConvertToTaxInvoice(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "tax invoice created", taxDoc)
}

func (h Handler) ConvertToDeliveryOrder(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	doc, err := h.service.ConvertToDeliveryOrder(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "delivery order created", doc)
}

func (h Handler) ConvertQuotation(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	doc, err := h.service.ConvertQuotation(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "invoice created from quotation", doc)
}

// Convert creates a new document of body.target_type from the source document,
// validated against the workflow matrix (allowedConversions).
func (h Handler) Convert(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	var req ConvertRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if req.TargetType == "" {
		return httpx.Error(c, fiber.StatusBadRequest, "target_type required", nil)
	}
	doc, err := h.service.Convert(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, req.TargetType)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "document converted", doc)
}

func (h Handler) RelatedDocuments(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	items, err := h.service.RelatedDocuments(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "related documents", fiber.Map{"items": items})
}

func (h Handler) BulkAction(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req BulkActionRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.service.BulkAction(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "bulk action completed", nil)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.Error(c, fiber.StatusNotFound, "not found", err.Error())
	case errors.Is(err, ErrForbidden):
		return httpx.Error(c, fiber.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrNoItems), errors.Is(err, ErrBadAction), errors.Is(err, ErrInvalidConversion):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal error", err.Error())
	}
}
