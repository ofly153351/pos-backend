package productbrand

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
	var req CreateProductBrandRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "product brand created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"))
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product brands fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	var req UpdateProductBrandRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("brandID"), req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product brand updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	if err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("brandID")); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product brand deleted", nil)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidName):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrDuplicateName):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrProductBrandNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	default:
		log.Printf("[productbrand] unhandled error: %v", err)
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
