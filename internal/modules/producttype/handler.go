package producttype

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) Create(c *fiber.Ctx) error {
	var req CreateProductTypeRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "product type created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"))
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product types fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	var req UpdateProductTypeRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("productTypeID"), req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product type updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	if err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("productTypeID")); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product type deleted", nil)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidName):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrDuplicateName):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrProductTypeNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	default:
		log.Printf("[producttype] unhandled error: %v", err)
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
