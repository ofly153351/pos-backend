package httpx

import (
	"github.com/gofiber/fiber/v2"
)

func DecodeJSON(c *fiber.Ctx, target any) error {
	return c.BodyParser(target)
}
