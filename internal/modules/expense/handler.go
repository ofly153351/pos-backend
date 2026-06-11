package expense

import (
	"errors"
	"log"
	"strconv"

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

// ── Expenses ────────────────────────────────────────────────────────────────

func (h Handler) List(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	query := ExpenseListQuery{
		From:          c.Query("from"),
		To:            c.Query("to"),
		CategoryID:    c.Query("category_id"),
		PaymentMethod: c.Query("payment_method"),
		Page:          page,
		Limit:         limit,
	}

	result, err := h.service.ListExpenses(c.UserContext(), middleware.ClaimsFromContext(c), storeID, query)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense list fetched", result)
}

func (h Handler) Summary(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.Summary(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense summary fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	expenseID := c.Params("expenseID")
	result, err := h.service.GetExpense(c.UserContext(), middleware.ClaimsFromContext(c), storeID, expenseID)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense fetched", result)
}

func (h Handler) Create(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateExpenseRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.CreateExpense(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "expense created", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	expenseID := c.Params("expenseID")
	var req UpdateExpenseRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.UpdateExpense(c.UserContext(), middleware.ClaimsFromContext(c), storeID, expenseID, req)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	expenseID := c.Params("expenseID")
	if err := h.service.VoidExpense(c.UserContext(), middleware.ClaimsFromContext(c), storeID, expenseID); err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense deleted", nil)
}

// ── Categories ──────────────────────────────────────────────────────────────

func (h Handler) ListCategories(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.ListCategories(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense categories fetched", result)
}

func (h Handler) CreateCategory(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateCategoryRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.CreateCategory(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "expense category created", result)
}

func (h Handler) UpdateCategory(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	categoryID := c.Params("categoryID")
	var req UpdateCategoryRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}

	result, err := h.service.UpdateCategory(c.UserContext(), middleware.ClaimsFromContext(c), storeID, categoryID, req)
	if err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense category updated", result)
}

func (h Handler) DeactivateCategory(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	categoryID := c.Params("categoryID")
	if err := h.service.DeactivateCategory(c.UserContext(), middleware.ClaimsFromContext(c), storeID, categoryID); err != nil {
		return writeExpenseError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "expense category deactivated", nil)
}

// ── Error mapping ───────────────────────────────────────────────────────────

func writeExpenseError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidExpenseDate):
		return httpx.Err422(c, "expense_date", err.Error())
	case errors.Is(err, ErrInvalidDescription):
		return httpx.Err422(c, "description", err.Error())
	case errors.Is(err, ErrInvalidAmount):
		return httpx.Err422(c, "amount", err.Error())
	case errors.Is(err, ErrInvalidPaymentMethod):
		return httpx.Err422(c, "payment_method", err.Error())
	case errors.Is(err, ErrInvalidCategory):
		return httpx.Err422(c, "category_id", err.Error())
	case errors.Is(err, ErrCategoryNameRequired):
		return httpx.Err422(c, "name", err.Error())
	case errors.Is(err, ErrCategoryDuplicate):
		return httpx.ErrConflict(c, err.Error())
	case errors.Is(err, ErrExpenseStoreIDRequired):
		return httpx.ErrBadRequest(c, err.Error())
	case errors.Is(err, ErrExpenseForbidden),
		errors.Is(err, ErrExpenseEditForbidden),
		errors.Is(err, ErrExpenseDeleteForbidden):
		return httpx.ErrForbidden(c, err.Error())
	case errors.Is(err, ErrExpenseNotFound), errors.Is(err, ErrCategoryNotFound):
		return httpx.ErrNotFound(c, err.Error())
	default:
		log.Printf("[expense] internal error: %v", err)
		return httpx.ErrInternal(c)
	}
}
