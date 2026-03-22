package subscription

import (
	"errors"

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

func (h Handler) ListPlans(c *fiber.Ctx) error {
	result, err := h.service.ListPlans(c.UserContext())
	if err != nil {
		return writeSubscriptionError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "subscription plans fetched", result)
}

func (h Handler) AdminListAll(c *fiber.Ctx) error {
	result, err := h.service.AdminListAll(c.UserContext())
	if err != nil {
		return writeSubscriptionError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "all store subscriptions fetched", result)
}

func (h Handler) GetCurrentByStore(c *fiber.Ctx) error {
	result, err := h.service.GetCurrentByStore(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"))
	if err != nil {
		return writeSubscriptionError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "store subscription fetched", result)
}

func (h Handler) ChangePlan(c *fiber.Ctx) error {
	var req ChangeSubscriptionRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.ChangePlan(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), req)
	if err != nil {
		return writeSubscriptionError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "store subscription updated", result)
}

func (h Handler) AdminChangePlan(c *fiber.Ctx) error {
	var req ChangeSubscriptionRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.AdminChangePlan(c.UserContext(), c.Params("storeID"), req)
	if err != nil {
		return writeSubscriptionError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "admin updated store subscription plan", result)
}

func (h Handler) AdminUpdateStatus(c *fiber.Ctx) error {
	var req UpdateSubscriptionStatusRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.AdminUpdateStatus(c.UserContext(), c.Params("storeID"), req)
	if err != nil {
		return writeSubscriptionError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "admin updated store subscription status", result)
}

func writeSubscriptionError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidPlanCode), errors.Is(err, ErrPlanNotFound), errors.Is(err, ErrInvalidStatus):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrSubscriptionNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
