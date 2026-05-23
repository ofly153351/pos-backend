package location

import (
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
	var req CreateLocationRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNameRequired, ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrLocationExists:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusCreated, "location created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	warehouseID := c.Query("warehouse_id")
	claims := middleware.ClaimsFromContext(c)

	result, err := h.service.ListByStore(c.UserContext(), claims, storeID, warehouseID)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "locations fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	claims := middleware.ClaimsFromContext(c)

	result, err := h.service.GetByID(c.UserContext(), claims, storeID, locationID)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "location fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	var req UpdateLocationRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), storeID, locationID, req)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrLocationExists:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "location updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	claims := middleware.ClaimsFromContext(c)

	err := h.service.Delete(c.UserContext(), claims, storeID, locationID)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrLocationInUse:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "location deleted", nil)
}
