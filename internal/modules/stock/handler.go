package stock

import (
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

func (h Handler) GetByProduct(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	productID := c.Params("productID")
	claims := middleware.ClaimsFromContext(c)

	result, err := h.service.GetByProduct(c.UserContext(), claims, storeID, productID)
	if err != nil {
		switch err {
		case ErrStockForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "stock summary fetched", result)
}

func (h Handler) GetByLocation(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	claims := middleware.ClaimsFromContext(c)

	result, err := h.service.GetByLocation(c.UserContext(), claims, storeID, locationID)
	if err != nil {
		switch err {
		case ErrStockForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "stock at location fetched", result)
}

func (h Handler) ListLowStock(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	claims := middleware.ClaimsFromContext(c)

	thresholdStr := c.Query("threshold", "5")
	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil || threshold < 0 {
		threshold = 5
	}

	result, err := h.service.ListLowStock(c.UserContext(), claims, storeID, threshold)
	if err != nil {
		switch err {
		case ErrStockForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "low stock items fetched", result)
}
