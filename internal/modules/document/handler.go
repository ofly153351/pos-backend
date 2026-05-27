package document

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) ListDocuments(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	q := ListQuery{
		StoreID:       storeID,
		Search:        c.Query("search"),
		Type:          c.Query("type"),
		Status:        c.Query("status"),
		PaymentStatus: c.Query("payment_status"),
		CustomerID:    c.Query("customer_id"),
		StaffID:       c.Query("staff_id"),
		DateFrom:      c.Query("date_from"),
		DateTo:        c.Query("date_to"),
		Page:          c.QueryInt("page", 1),
		Limit:         c.QueryInt("limit", 20),
	}
	result, err := h.service.ListDocuments(c.UserContext(), middleware.ClaimsFromContext(c), q)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "documents fetched", result)
}

func (h Handler) GetDocument(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	doc, err := h.service.GetDocument(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "document fetched", doc)
}

func (h Handler) CreateDocument(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	doc, err := h.service.CreateDocument(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "document created", doc)
}

func (h Handler) UpdateDocumentStatus(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	var req UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.service.UpdateDocumentStatus(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id, req); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "status updated", nil)
}

func (h Handler) DeleteDocument(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	id := c.Params("docID")
	if err := h.service.DeleteDocument(c.UserContext(), middleware.ClaimsFromContext(c), storeID, id); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "document deleted", nil)
}

func (h Handler) BulkAction(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req BulkActionRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.service.BulkAction(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "bulk action completed", nil)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.Error(c, fiber.StatusNotFound, "not found", err.Error())
	case errors.Is(err, ErrForbidden):
		return httpx.Error(c, fiber.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrNoItems), errors.Is(err, ErrBadAction):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal error", err.Error())
	}
}
