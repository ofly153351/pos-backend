package store

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

func (h Handler) Create(c *fiber.Ctx) error {
	request := CreateStoreRequest{
		Name:                 c.FormValue("name"),
		Phone:                c.FormValue("phone"),
		Address:              c.FormValue("address"),
		CurrencyCode:         c.FormValue("currency_code"),
		SubscriptionPlanCode: c.FormValue("subscription_plan_code"),
	}

	logo, err := c.FormFile("logo")
	if err == nil {
		request.LogoFile = logo
	}

	result, err := h.service.CreateStore(c.UserContext(), middleware.ClaimsFromContext(c), request)
	if err != nil {
		return writeStoreError(c, err)
	}

	return httpx.Success(c, fiber.StatusCreated, "store created", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	if storeID == "" {
		return httpx.Error(c, fiber.StatusBadRequest, "storeID is required", nil)
	}

	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeStoreError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "store fetched", result)
}

func writeStoreError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidStoreName), errors.Is(err, ErrInvalidCurrencyCode), errors.Is(err, ErrInvalidSubscriptionPlan):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrStoreForbidden):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrStoreNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	case errors.Is(err, ErrSubscriptionPlanNotFound):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
