package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/httpx"
)

const claimsKey = "auth_claims"

func AuthRequired(tokens auth.TokenManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := strings.TrimSpace(c.Get("Authorization"))
		if header == "" {
			return httpx.Error(c, fiber.StatusUnauthorized, "missing authorization header", nil)
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return httpx.Error(c, fiber.StatusUnauthorized, "invalid authorization header", nil)
		}

		claims, err := tokens.Parse(parts[1])
		if err != nil {
			return httpx.Error(c, fiber.StatusUnauthorized, "invalid or expired token", nil)
		}

		c.Locals(claimsKey, claims)
		return c.Next()
	}
}

func ClaimsFromContext(c *fiber.Ctx) auth.Claims {
	claims, _ := c.Locals(claimsKey).(auth.Claims)
	return claims
}

func RequireRoles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := ClaimsFromContext(c)
		for _, role := range roles {
			if claims.Role == role {
				return c.Next()
			}
		}

		return httpx.Error(c, fiber.StatusForbidden, "forbidden", nil)
	}
}
