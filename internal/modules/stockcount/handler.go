package stockcount

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/stock_movement"
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
	return httpx.Success(c, fiber.StatusOK, "stock count sessions fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	sessionID := c.Params("sessionID")
	result, err := h.service.Get(c.UserContext(), middleware.ClaimsFromContext(c), storeID, sessionID)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "stock count session fetched", result)
}

func (h Handler) Save(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	sessionID := c.Params("sessionID")
	var req CountSession
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Save(c.UserContext(), middleware.ClaimsFromContext(c), storeID, sessionID, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "stock count session saved", result)
}

func (h Handler) Apply(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	sessionID := c.Params("sessionID")
	var req ApplyRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Apply(c.UserContext(), middleware.ClaimsFromContext(c), storeID, sessionID, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "stock count applied", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	sessionID := c.Params("sessionID")
	if err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), storeID, sessionID); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "stock count session deleted", nil)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrStoreIDRequired), errors.Is(err, ErrSessionIDRequired):
		return httpx.ErrBadRequest(c, err.Error())
	case errors.Is(err, ErrSessionIDMismatch):
		return httpx.Err422(c, "id", err.Error())
	case errors.Is(err, ErrForbidden):
		return httpx.ErrForbidden(c, err.Error())
	case errors.Is(err, ErrSessionNotFound):
		return httpx.ErrNotFound(c, err.Error())
	case errors.Is(err, ErrNoItemsToApply), errors.Is(err, ErrInvalidApplyItem):
		return httpx.Err422(c, "items", err.Error())
	case errors.Is(err, ErrSessionAlreadyApplied):
		return httpx.ErrConflict(c, err.Error())
	case errors.Is(err, ErrCountLocationRequired):
		// Legacy / location-less session cannot be applied (aggregate → single-location unsafe).
		return httpx.ErrConflict(c, err.Error())
	case errors.Is(err, ErrCountLocationInvalid):
		return httpx.Err422(c, "location", err.Error())
	case errors.Is(err, stock_movement.ErrInsufficientStock):
		return httpx.Err422(c, "stock", err.Error())
	case errors.Is(err, stock_movement.ErrStockStaleCount):
		// A counted quantity no longer matches the location's live on-hand (multi-location
		// total, or stock moved after counting). Surface as a conflict so the worksheet can
		// be re-counted per location rather than overwriting one location with a total.
		return httpx.ErrConflict(c, err.Error())
	case errors.Is(err, stock_movement.ErrStockExpectedRequired):
		return httpx.Err422(c, "items", err.Error())
	case errors.Is(err, stock_movement.ErrProductNotFound):
		return httpx.ErrNotFound(c, err.Error())
	case errors.Is(err, stock_movement.ErrStockForbidden):
		return httpx.ErrForbidden(c, err.Error())
	default:
		log.Printf("[stockcount] internal error: %v", err)
		return httpx.ErrInternal(c)
	}
}
