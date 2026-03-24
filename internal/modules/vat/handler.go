package vat

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) Calculate(c *fiber.Ctx) error {
	var req CalculateVATRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.Calculate(req)
	if err != nil {
		return writeVATError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "vat calculated", fiber.Map{"summary": result})
}

func writeVATError(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrNoItemsProvided) {
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
}
