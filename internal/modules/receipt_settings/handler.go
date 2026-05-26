package receipt_settings

import (
	"errors"

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

func (h Handler) Get(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	if storeID == "" {
		return httpx.Error(c, fiber.StatusBadRequest, "storeID is required", nil)
	}

	result, err := h.service.Get(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "receipt settings fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	if storeID == "" {
		return httpx.Error(c, fiber.StatusBadRequest, "storeID is required", nil)
	}

	var req UpdateReceiptSettingsRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "receipt settings updated", result)
}

func (h Handler) PreviewHTML(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	if storeID == "" {
		return httpx.Error(c, fiber.StatusBadRequest, "storeID is required", nil)
	}

	var req UpdateReceiptSettingsRequest
	// Body is optional — empty body means "preview current saved settings"
	_ = httpx.DecodeJSON(c, &req)

	html, err := h.service.PreviewHTML(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeError(c, err)
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(html)
}

func writeError(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrForbidden) {
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	}
	if errors.Is(err, ErrNotFound) {
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	}
	return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
}
