package promotion

import (
	"errors"
	"log"

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

func (h Handler) List(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.List(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "promotions fetched", result)
}

func (h Handler) Create(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), storeID, c.Body())
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "promotion created", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("promotionID")
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, c.Body())
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "promotion updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("promotionID")
	if err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "promotion deleted", nil)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrStoreIDRequired), errors.Is(err, ErrIDRequired):
		return httpx.ErrBadRequest(c, err.Error())
	case errors.Is(err, ErrInvalidBody):
		return httpx.Err422(c, "body", err.Error())
	case errors.Is(err, ErrForbidden):
		return httpx.ErrForbidden(c, err.Error())
	case errors.Is(err, ErrNotFound):
		return httpx.ErrNotFound(c, err.Error())
	default:
		log.Printf("[promotion] internal error: %v", err)
		return httpx.ErrInternal(c)
	}
}
