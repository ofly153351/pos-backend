package store

import (
	"errors"
	"log"
	"strings"

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
		PromptPayID:          c.FormValue("promptpay_id"),
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

func (h Handler) ListMyStores(c *fiber.Ctx) error {
	result, err := h.service.ListMyStores(c.UserContext(), middleware.ClaimsFromContext(c))
	if err != nil {
		return writeStoreError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "stores fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	if storeID == "" {
		return httpx.Error(c, fiber.StatusBadRequest, "storeID is required", nil)
	}

	request, err := parseUpdateStoreRequest(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), storeID, request)
	if err != nil {
		return writeStoreError(c, err)
	}

	return httpx.Success(c, fiber.StatusOK, "store updated", result)
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
		claims := middleware.ClaimsFromContext(c)
		log.Printf(
			"[store] internal error: method=%s path=%s store_id=%s user_id=%s role=%s err=%v",
			c.Method(),
			c.Path(),
			c.Params("storeID"),
			claims.UserID,
			claims.Role,
			err,
		)
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}

func parseUpdateStoreRequest(c *fiber.Ctx) (UpdateStoreRequest, error) {
	request := UpdateStoreRequest{}
	if strings.HasPrefix(c.Get(fiber.HeaderContentType), fiber.MIMEApplicationJSON) {
		if err := httpx.DecodeJSON(c, &request); err != nil {
			return UpdateStoreRequest{}, err
		}
		return request, nil
	}

	if value := c.FormValue("name"); value != "" {
		request.Name = &value
	}
	if value := c.FormValue("phone"); value != "" {
		request.Phone = &value
	}
	if value := c.FormValue("fax"); value != "" {
		request.Fax = &value
	}
	if value := c.FormValue("email"); value != "" {
		request.Email = &value
	}
	if value := c.FormValue("website"); value != "" {
		request.Website = &value
	}
	if value := c.FormValue("address"); value != "" {
		request.Address = &value
	}
	if value := c.FormValue("tax_id"); value != "" {
		request.TaxID = &value
	}
	if value := c.FormValue("promptpay_id"); value != "" {
		request.PromptPayID = &value
	}
	if value := c.FormValue("currency_code"); value != "" {
		request.CurrencyCode = &value
	}

	logo, err := c.FormFile("logo")
	if err == nil {
		request.LogoFile = logo
	}

	return request, nil
}
