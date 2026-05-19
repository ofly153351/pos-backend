package stock_movement

import (
	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
	"strconv"
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
		case ErrStockNoItems, ErrStockBadQty:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrProductNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusCreated, "stock added", result)
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

	role := claims.Role
	userID := claims.UserID

	// Check store access - platform_admin can always access
	if role != "platform_admin" {
		// The service/repository doesn't check user access directly for list
		// We rely on the store-scoped query
		_ = userID
	}

	result, err := h.service.ListMovements(c.UserContext(), claims, storeID, q)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
	return httpx.Success(c, fiber.StatusOK, "movements fetched", result)
}
