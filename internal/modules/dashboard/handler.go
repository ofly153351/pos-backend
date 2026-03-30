package dashboard

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

func (h Handler) GetOverview(c *fiber.Ctx) error {
	query, err := parseOverviewQuery(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	result, err := h.service.GetOverview(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), query)
	if err != nil {
		return writeDashboardError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "dashboard fetched", result)
}

func writeDashboardError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidPeriod), errors.Is(err, ErrInvalidTimeRange), errors.Is(err, ErrInvalidLimit):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}

func parseOverviewQuery(c *fiber.Ctx) (OverviewQuery, error) {
	query := OverviewQuery{
		Period: strings.TrimSpace(c.Query("period")),
	}

	if text := strings.TrimSpace(c.Query("from")); text != "" {
		parsed, err := parseDateTime(text, false)
		if err != nil {
			return OverviewQuery{}, ErrInvalidTimeRange
		}
		query.From = &parsed
	}
	if text := strings.TrimSpace(c.Query("to")); text != "" {
		parsed, err := parseDateTime(text, true)
		if err != nil {
			return OverviewQuery{}, ErrInvalidTimeRange
		}
		query.To = &parsed
	}

	var err error
	if raw := strings.TrimSpace(c.Query("top_limit")); raw != "" {
		query.TopLimit, err = strconv.Atoi(raw)
		if err != nil {
			return OverviewQuery{}, ErrInvalidLimit
		}
	}
	if raw := strings.TrimSpace(c.Query("recent_limit")); raw != "" {
		query.RecentLimit, err = strconv.Atoi(raw)
		if err != nil {
			return OverviewQuery{}, ErrInvalidLimit
		}
	}
	if raw := strings.TrimSpace(c.Query("low_stock_limit")); raw != "" {
		query.LowStockLimit, err = strconv.Atoi(raw)
		if err != nil {
			return OverviewQuery{}, ErrInvalidLimit
		}
	}
	if raw := strings.TrimSpace(c.Query("low_stock_threshold")); raw != "" {
		query.LowStockThreshold, err = strconv.Atoi(raw)
		if err != nil {
			return OverviewQuery{}, ErrInvalidLimit
		}
	}

	return query, nil
}

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
