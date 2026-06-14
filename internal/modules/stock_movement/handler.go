package stock_movement

import (
	"strconv"
	"strings"

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

func (h Handler) AddStock(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req AddStockRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AddStock(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		switch err {
		case ErrStockNoItems, ErrStockBadQty, ErrStockLocationRequired,
			ErrStockReasonRequired, ErrStockReasonInvalid, ErrStockReasonNoteRequired:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrProductNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrStockIdempotencyConflict:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		case ErrStockForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusCreated, "stock added", result)
}

func (h Handler) RemoveStock(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req RemoveStockRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.RemoveStock(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		switch err {
		case ErrStockBadQty, ErrStockLocationRequired,
			ErrStockReasonRequired, ErrStockReasonInvalid, ErrStockReasonNoteRequired:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrProductNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrInsufficientStock, ErrStockExceedsAvailable, ErrStockIdempotencyConflict:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		case ErrStockForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "stock removed", result)
}

func (h Handler) TransferStock(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req TransferStockRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	// Idempotency key (Phase W4A §9): header takes precedence; a network-retried transfer
	// with the same key returns the original result instead of moving stock twice.
	if hk := strings.TrimSpace(c.Get("Idempotency-Key")); hk != "" {
		req.IdempotencyKey = hk
	}
	result, err := h.service.TransferStock(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		switch err {
		case ErrStockBadQty, ErrTransferSourceRequired, ErrTransferDestRequired,
			ErrTransferSameLocation, ErrTransferCrossStore, ErrTransferLocationNotInStore, ErrTransferLocationInactive,
			ErrStockReasonRequired, ErrStockReasonInvalid, ErrStockReasonNoteRequired:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrProductNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrTransferInsufficient, ErrTransferIdempotencyConflict:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		case ErrStockForbidden, ErrTransferForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "stock transferred", result)
}

func (h Handler) AdjustStock(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req AdjustStockRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AdjustStock(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		switch err {
		case ErrStockBadQty, ErrStockLocationRequired,
			ErrStockReasonRequired, ErrStockReasonInvalid, ErrStockReasonNoteRequired:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrProductNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrStockNoChange, ErrStockIdempotencyConflict:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		case ErrStockForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "stock adjusted", result)
}

func (h Handler) ListMovements(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	claims := middleware.ClaimsFromContext(c)

	productID := c.Query("product_id")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	q := ListMovementsQuery{
		ProductID: productID,
		Page:      page,
		Limit:     limit,
	}

	result, err := h.service.ListMovements(c.UserContext(), claims, storeID, q)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
	return httpx.Success(c, fiber.StatusOK, "movements fetched", result)
}
