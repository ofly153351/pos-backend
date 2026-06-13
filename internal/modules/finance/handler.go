package finance

import (
	"errors"
	"strconv"
	"strings"
	"time"

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

func (h Handler) GetPnL(c *fiber.Ctx) error {
	query, err := parsePnLQuery(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	result, err := h.service.GetPnL(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), query)
	if err != nil {
		return writeFinanceError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "pnl fetched", result)
}

func (h Handler) GetSummary(c *fiber.Ctx) error {
	query, err := parsePnLQuery(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	result, err := h.service.GetSummary(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), query)
	if err != nil {
		return writeFinanceError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "summary fetched", result)
}

func (h Handler) GetInventory(c *fiber.Ctx) error {
	// dead_days = idle threshold for dead-stock (30/60/90 from the report filter);
	// invalid/absent falls back to the service default (30).
	deadDays, _ := strconv.Atoi(strings.TrimSpace(c.Query("dead_days")))

	result, err := h.service.GetInventoryReport(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), deadDays)
	if err != nil {
		return writeFinanceError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "inventory report fetched", result)
}

func writeFinanceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidPeriod), errors.Is(err, ErrInvalidTimeRange):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}

func parsePnLQuery(c *fiber.Ctx) (PnLQuery, error) {
	query := PnLQuery{
		Period: strings.TrimSpace(c.Query("period")),
	}

	if text := strings.TrimSpace(c.Query("from")); text != "" {
		parsed, err := parseDateTime(text, false)
		if err != nil {
			return PnLQuery{}, ErrInvalidTimeRange
		}
		query.From = &parsed
	}
	if text := strings.TrimSpace(c.Query("to")); text != "" {
		parsed, err := parseDateTime(text, true)
		if err != nil {
			return PnLQuery{}, ErrInvalidTimeRange
		}
		query.To = &parsed
	}

	return query, nil
}

// parseDateTime accepts RFC3339 or a bare YYYY-MM-DD; endOfDay pushes a bare date
// to the next midnight so the range stays half-open and inclusive of the to-day.
func parseDateTime(text string, endOfDay bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse("2006-01-02", text)
	if err != nil {
		return time.Time{}, err
	}
	parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	if endOfDay {
		parsed = parsed.Add(24 * time.Hour)
	}
	return parsed, nil
}
