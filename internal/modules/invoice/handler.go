package invoice

import (
	"errors"
	"fmt"
	"strconv"
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
	var req CreateInvoiceRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), req)
	if err != nil {
		return writeInvoiceError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "invoice created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"))
	if err != nil {
		return writeInvoiceError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "invoices fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("invoiceID"))
	if err != nil {
		return writeInvoiceError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "invoice fetched", result)
}

func (h Handler) AddPayment(c *fiber.Ctx) error {
	req, err := parseAddPaymentRequest(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AddPayment(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("invoiceID"), req)
	if err != nil {
		return writeInvoiceError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "invoice payment recorded", result)
}

func (h Handler) ExportPDF(c *fiber.Ctx) error {
	content, err := h.service.GeneratePDF(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("invoiceID"))
	if err != nil {
		return writeInvoiceError(c, err)
	}
	c.Set(fiber.HeaderContentType, "application/pdf")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`inline; filename="%s.pdf"`, c.Params("invoiceID")))
	return c.Status(fiber.StatusOK).Send(content)
}

func writeInvoiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInvoiceItems), errors.Is(err, ErrInvalidInvoiceItem), errors.Is(err, ErrInvalidPaidAmount), errors.Is(err, ErrInvalidPaymentMethod), errors.Is(err, ErrInvalidProofFileType), errors.Is(err, ErrInvalidProofFileSize), errors.Is(err, ErrInvalidDiscountType), errors.Is(err, ErrDiscountValueRequired), errors.Is(err, ErrInvalidDiscountValue), errors.Is(err, ErrInvalidPercentDiscount), errors.Is(err, ErrAmountDiscountExceeds), errors.Is(err, ErrProductNotFound), errors.Is(err, ErrProductInactive), errors.Is(err, ErrInsufficientStock), errors.Is(err, ErrInvoiceAlreadyPaid), errors.Is(err, ErrPaymentExceedsRemaining):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrInvoiceNotFound), errors.Is(err, ErrCustomerNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}

func parseAddPaymentRequest(c *fiber.Ctx) (CreateInvoicePaymentRequest, error) {
	contentType := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	if strings.Contains(contentType, "multipart/form-data") {
		var req CreateInvoicePaymentRequest
		paidAmount, err := strconv.ParseFloat(strings.TrimSpace(c.FormValue("paid_amount")), 64)
		if err != nil {
			return CreateInvoicePaymentRequest{}, err
		}
		req.PaidAmount = paidAmount
		req.PaymentMethod = c.FormValue("payment_method")
		req.Note = c.FormValue("note")
		file, err := c.FormFile("proof")
		if err == nil {
			req.ProofFile = file
		}
		return req, nil
	}

	var req CreateInvoicePaymentRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return CreateInvoicePaymentRequest{}, err
	}
	return req, nil
}
