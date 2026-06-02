package activity_log

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/platform/httpx"
)

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

	result, err := h.service.List(c.UserContext(), ListQuery{
		StoreID:  c.Params("storeID"),
		Module:   c.Query("module"),
		Action:   c.Query("action"),
		UserID:   c.Query("user_id"),
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "failed to fetch activity logs", err.Error())
	}
	return httpx.Success(c, fiber.StatusOK, "activity logs fetched", result)
}
