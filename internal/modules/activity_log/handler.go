package activity_log

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/httpx"
)

// claimsLocalKey mirrors middleware.claimsKey. It is duplicated (not imported)
// because the middleware package imports this one for action logging — importing
// middleware back would create an import cycle.
const claimsLocalKey = "auth_claims"

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) Service() Service { return h.service }

func (h Handler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	claims, _ := c.Locals(claimsLocalKey).(auth.Claims)
	result, err := h.service.List(c.UserContext(), claims, ListQuery{
		StoreID:    c.Params("storeID"),
		Module:     c.Query("module"),
		Action:     c.Query("action"),
		UserID:     c.Query("user_id"),
		ResourceID: c.Query("resource_id"),
		Severity:   c.Query("severity"),
		Category:   c.Query("category"),
		Search:     c.Query("q"),
		DateFrom:   c.Query("date_from"),
		DateTo:     c.Query("date_to"),
		Page:       page,
		Limit:      limit,
	})
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			return httpx.ErrForbidden(c, err.Error())
		}
		return httpx.Error(c, fiber.StatusInternalServerError, "failed to fetch activity logs", err.Error())
	}
	return httpx.Success(c, fiber.StatusOK, "activity logs fetched", result)
}
