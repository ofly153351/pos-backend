package warehouse_dashboard

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

// GetDashboard handles GET /stores/:storeID/dashboard/warehouse
func (h Handler) GetDashboard(c *fiber.Ctx) error {
	q, err := parseQuery(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	data, err := h.service.GetDashboard(
		c.UserContext(),
		middleware.ClaimsFromContext(c),
		c.Params("storeID"),
		q,
	)
	if err != nil {
		return writeError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "warehouse dashboard fetched", data)
}

func parseQuery(c *fiber.Ctx) (Query, error) {
	raw := c.Query("period", string(Period7d))
	switch Period(raw) {
	case Period7d, Period30d, Period3m:
		return Query{Period: Period(raw)}, nil
	default:
		return Query{}, ErrInvalidPeriod
	}
}

func writeError(c *fiber.Ctx, err error) error {
	switch err {
	case ErrForbidden:
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case ErrInvalidPeriod:
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", err.Error())
	}
}
