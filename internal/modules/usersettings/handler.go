package usersettings

import (
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

// GetCardSettings → GET /me/card-settings
func (h Handler) GetCardSettings(c *fiber.Ctx) error {
	userID := middleware.ClaimsFromContext(c).UserID
	result, err := h.service.GetCardSettings(c.UserContext(), userID)
	if err != nil {
		log.Printf("[usersettings] get error: %v", err)
		return httpx.ErrInternal(c)
	}
	return httpx.Success(c, fiber.StatusOK, "card settings fetched", result)
}

// UpdateCardSettings → PUT /me/card-settings
func (h Handler) UpdateCardSettings(c *fiber.Ctx) error {
	var req CardSettings
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.ErrBadRequest(c, "invalid request body")
	}
	userID := middleware.ClaimsFromContext(c).UserID
	result, err := h.service.SaveCardSettings(c.UserContext(), userID, req)
	if err != nil {
		log.Printf("[usersettings] save error: %v", err)
		return httpx.ErrInternal(c)
	}
	return httpx.Success(c, fiber.StatusOK, "card settings saved", result)
}
