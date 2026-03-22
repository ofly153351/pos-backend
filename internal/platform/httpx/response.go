package httpx

import (
	"github.com/gofiber/fiber/v2"
)

type envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func JSON(c *fiber.Ctx, status int, payload any) error {
	return c.Status(status).JSON(payload)
}

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return JSON(c, status, envelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *fiber.Ctx, status int, message string, detail any) error {
	return JSON(c, status, envelope{
		Success: false,
		Message: message,
		Error:   detail,
	})
}
