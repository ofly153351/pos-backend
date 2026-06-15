package sale

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/customer"
	"pos-backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) Create(c *fiber.Ctx) error {
	var req CreateSaleRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	// Phase W4B — request idempotency travels in the header, never the JSON body.
	req.IdempotencyKey = c.Get("Idempotency-Key")

	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), req)
	if err != nil {
		return writeSaleError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "sale created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"))
	if err != nil {
		return writeSaleError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "sales fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("saleID"))
	if err != nil {
		return writeSaleError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "sale fetched", result)
}

func (h Handler) Receipt(c *fiber.Ctx) error {
	content, err := h.service.GenerateReceiptHTML(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("saleID"))
	if err != nil {
		return writeSaleError(c, err)
	}

	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.Status(fiber.StatusOK).Send(content)
}

func (h Handler) ReceiptPreview(c *fiber.Ctx) error {
	content, err := h.service.GenerateReceiptPreviewHTML(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("saleID"))
	if err != nil {
		return writeSaleError(c, err)
	}

	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.Status(fiber.StatusOK).Send(content)
}

func writeSaleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidSaleItems), errors.Is(err, ErrInvalidSaleItem):
		return httpx.Err422(c, "items", err.Error())
	case errors.Is(err, ErrInvalidPaymentMethod):
		return httpx.Err422(c, "payment_method", err.Error())
	case errors.Is(err, ErrInvalidPaidAmount):
		return httpx.Err422(c, "paid_amount", err.Error())
	case errors.Is(err, ErrInvalidBillDiscount), errors.Is(err, ErrBillDiscountExceedsAmount):
		return httpx.Err422(c, "bill_discount_value", err.Error())
	case errors.Is(err, ErrManualDiscountExceedsCap):
		return httpx.Err422(c, "manual_discount", err.Error())
	case errors.Is(err, ErrInvalidDiscountType):
		return httpx.Err422(c, "discount_type", err.Error())
	case errors.Is(err, ErrDiscountValueRequired), errors.Is(err, ErrInvalidDiscountValue),
		errors.Is(err, ErrInvalidPercentDiscount), errors.Is(err, ErrAmountDiscountExceedsPrice):
		return httpx.Err422(c, "discount_value", err.Error())
	case errors.Is(err, ErrInvalidVATPercent):
		return httpx.Err422(c, "vat_percent", err.Error())
	case errors.Is(err, ErrProductInactive):
		return httpx.ErrConflict(c, err.Error())
	case errors.Is(err, ErrInsufficientStock):
		// Includes InsufficientSaleStockError (unwraps to ErrInsufficientStock); its
		// Error() already carries the Thai shortfall message with the on-hand count.
		return httpx.ErrConflict(c, err.Error())
	case errors.Is(err, ErrNoSaleLocation), errors.Is(err, ErrSaleLocationInvalid),
		errors.Is(err, ErrSaleLocationCrossStore):
		return httpx.Err422(c, "location_id", err.Error())
	case errors.Is(err, ErrSaleIdempotencyConflict):
		return httpx.ErrConflict(c, err.Error())
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.ErrForbidden(c, err.Error())
	case errors.Is(err, ErrSaleNotFound), errors.Is(err, ErrCustomerNotFound),
		errors.Is(err, customer.ErrCustomerNotFound), errors.Is(err, ErrProductNotFound):
		return httpx.ErrNotFound(c, err.Error())
	default:
		log.Printf("[sale] internal error: %v", err)
		return httpx.ErrInternal(c)
	}
}
