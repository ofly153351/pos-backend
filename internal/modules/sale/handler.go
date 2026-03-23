package sale

import (
	"errors"

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

func writeSaleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidSaleItems), errors.Is(err, ErrInvalidSaleItem), errors.Is(err, ErrInvalidPaymentMethod), errors.Is(err, ErrInvalidPaidAmount), errors.Is(err, ErrInvalidDiscountType), errors.Is(err, ErrDiscountValueRequired), errors.Is(err, ErrInvalidDiscountValue), errors.Is(err, ErrInvalidPercentDiscount), errors.Is(err, ErrAmountDiscountExceedsPrice), errors.Is(err, ErrProductNotFound), errors.Is(err, ErrProductInactive), errors.Is(err, ErrInsufficientStock):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrSaleNotFound), errors.Is(err, ErrCustomerNotFound), errors.Is(err, customer.ErrCustomerNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
