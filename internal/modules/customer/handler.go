package customer

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) Create(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateCustomerRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "customer created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "customer list fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	customerID := c.Params("customerID")
	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), storeID, customerID)
	if err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "customer fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	customerID := c.Params("customerID")
	var req UpdateCustomerRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), storeID, customerID, req)
	if err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "customer updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	customerID := c.Params("customerID")
	if err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), storeID, customerID); err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "customer deleted", nil)
}

func (h Handler) ListLevelDiscounts(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.ListLevelDiscounts(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "customer level discounts fetched", result)
}

func (h Handler) UpsertLevelDiscount(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	level, err := strconv.Atoi(c.Params("level"))
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, ErrInvalidLevel.Error(), nil)
	}
	var req UpsertLevelDiscountRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.UpsertLevelDiscount(c.UserContext(), middleware.ClaimsFromContext(c), storeID, level, req)
	if err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "customer level discount upserted", result)
}

func (h Handler) DeleteLevelDiscount(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	level, err := strconv.Atoi(c.Params("level"))
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, ErrInvalidLevel.Error(), nil)
	}
	if err := h.service.DeleteLevelDiscount(c.UserContext(), middleware.ClaimsFromContext(c), storeID, level); err != nil {
		return writeCustomerError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "customer level discount deleted", nil)
}

func writeCustomerError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidCustomerName), errors.Is(err, ErrInvalidCustomerEmail), errors.Is(err, ErrCustomerStoreIDRequired), errors.Is(err, ErrInvalidLevel), errors.Is(err, ErrInvalidDiscountPercent):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrCustomerForbidden):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrCustomerNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
