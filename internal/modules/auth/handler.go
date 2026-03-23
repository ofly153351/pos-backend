package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.Register(c.UserContext(), req)
	if err != nil {
		return writeAuthError(c, err)
	}

	return httpx.Success(c, fiber.StatusCreated, "register success", result)
}

func (h Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.Login(c.UserContext(), req)
	if err != nil {
		return writeAuthError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "login success", result)
}

func (h Handler) Logout(c *fiber.Ctx) error {
	claims, _ := c.Locals("auth_claims").(Claims)
	if err := h.service.Logout(c.UserContext(), claims); err != nil {
		return writeAuthError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "logout success", nil)
}

func writeAuthError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrEmailExists):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUserNotFound):
		return httpx.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidPassword), errors.Is(err, ErrInvalidRole):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
