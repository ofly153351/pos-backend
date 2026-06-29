package creditsale

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/sale"
	"pos-backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) List(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.List(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "credit sales fetched", result)
}

func (h Handler) Summary(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.Summary(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "credit summary fetched", result)
}

func (h Handler) Aging(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.Aging(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "aging summary fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("creditSaleID")
	result, err := h.service.Get(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "credit sale fetched", result)
}

func (h Handler) Create(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateCreditSaleRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	req.IdempotencyKey = c.Get("Idempotency-Key")
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "credit sale created", result)
}

func (h Handler) AddPayment(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("creditSaleID")
	var req AddPaymentRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AddPayment(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "payment recorded", result)
}

func (h Handler) Cancel(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("creditSaleID")
	result, err := h.service.Cancel(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "credit sale cancelled", result)
}

// ReturnGoods restocks borrowed goods (loan type) and settles the receivable by their
// value. Body: { "items": [{ "product_id": "...", "quantity": N }] }.
func (h Handler) ReturnGoods(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("creditSaleID")
	var req ReturnGoodsRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.ReturnGoods(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "goods returned", result)
}

func (h Handler) Statement(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("creditSaleID")
	data, err := h.service.Statement(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "inline; filename=\"credit-statement.pdf\"")
	c.Set("Cache-Control", "no-store")
	return c.Send(data)
}

// Bill renders the credit sale as a ใบวางบิล (BILL) HTML document via the shared
// unified template — served inline so the frontend can preview it in a modal/iframe
// (no new tab, no PDF). Replaces the legacy client-side A4 print page.
func (h Handler) Bill(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("creditSaleID")
	html, err := h.service.BillHTML(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-store")
	return c.Status(fiber.StatusOK).SendString(html)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrStoreIDRequired):
		return httpx.ErrBadRequest(c, err.Error())
	case errors.Is(err, ErrForbidden):
		return httpx.ErrForbidden(c, err.Error())
	case errors.Is(err, ErrNotFound):
		return httpx.ErrNotFound(c, err.Error())
	case errors.Is(err, ErrCustomerRequired):
		return httpx.Err422(c, "customer_id", err.Error())
	case errors.Is(err, ErrNoItems), errors.Is(err, ErrInvalidItem), errors.Is(err, ErrNoReturnItems), errors.Is(err, ErrReturnExceedsLent):
		return httpx.Err422(c, "items", err.Error())
	case errors.Is(err, ErrInvalidType), errors.Is(err, ErrNotALoan):
		return httpx.Err422(c, "type", err.Error())
	case errors.Is(err, ErrInvalidDownPayment), errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrOverpayment):
		return httpx.Err422(c, "amount", err.Error())
	case errors.Is(err, ErrAlreadyCancelled), errors.Is(err, ErrPaymentAfterCancel), errors.Is(err, ErrCannotCancelPaid), errors.Is(err, ErrReturnAfterCancel):
		return httpx.ErrConflict(c, err.Error())
	// Errors bubbling up from the underlying sale create (client-fixable):
	case errors.Is(err, sale.ErrInsufficientStock):
		return httpx.Err422(c, "items", "insufficient sale-point stock for one or more items")
	case errors.Is(err, sale.ErrProductInactive):
		return httpx.Err422(c, "items", "one or more products are inactive")
	case errors.Is(err, sale.ErrForbiddenStoreAccess):
		return httpx.ErrForbidden(c, err.Error())
	default:
		log.Printf("[creditsale] internal error: %v", err)
		return httpx.ErrInternal(c)
	}
}
