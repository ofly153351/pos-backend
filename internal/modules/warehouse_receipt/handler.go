package warehouse_receipt

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) List(c *fiber.Ctx) error {
	page, limit, err := parsePaginationQuery(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, ErrInvalidPagination.Error(), err.Error())
	}
	status, err := parseStatusQuery(c)
	if err != nil {
		return writeReceiptError(c, err)
	}
	storeID := strings.TrimSpace(c.Query("store_id"))
	if storeID == "" {
		storeID = strings.TrimSpace(c.Params("storeID"))
	}
	result, err := h.service.List(c.UserContext(), middleware.ClaimsFromContext(c), ListReceiptsQuery{
		StoreID: storeID,
		Status:  status,
		Page:    page,
		Limit:   limit,
	})
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipts fetched", result)
}

func (h Handler) Create(c *fiber.Ctx) error {
	var req CreateReceiptRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), req)
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "warehouse receipt created", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	var req UpdateReceiptRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"), req)
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt updated", result)
}

func (h Handler) AddItems(c *fiber.Ctx) error {
	var req AddReceiptItemsRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AddItems(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"), req)
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt items saved", result)
}

func (h Handler) UpdateItem(c *fiber.Ctx) error {
	var req UpdateReceiptItemRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.UpdateItem(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"), c.Params("item_id"), req)
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt item updated", result)
}

func (h Handler) DeleteItem(c *fiber.Ctx) error {
	result, err := h.service.DeleteItem(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"), c.Params("item_id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt item deleted", result)
}

func (h Handler) Confirm(c *fiber.Ctx) error {
	result, err := h.service.Confirm(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt confirmed", result)
}

func (h Handler) Submit(c *fiber.Ctx) error {
	result, err := h.service.Submit(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt submitted for approval", result)
}

func (h Handler) Reopen(c *fiber.Ctx) error {
	result, err := h.service.Reopen(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt reopened to draft", result)
}

func (h Handler) Cancel(c *fiber.Ctx) error {
	result, err := h.service.Cancel(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt cancelled", result)
}

func (h Handler) UploadAttachment(c *fiber.Ctx) error {
	contentType := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	if !strings.Contains(contentType, "multipart/form-data") {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", "content-type must be multipart/form-data")
	}
	file, err := c.FormFile("file")
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, ErrReceiptAttachmentRequired.Error(), nil)
	}
	if strings.TrimSpace(file.Filename) == "" {
		return httpx.Error(c, fiber.StatusBadRequest, ErrReceiptAttachmentRequired.Error(), nil)
	}
	result, err := h.service.UploadAttachment(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"), UploadAttachmentRequest{File: file})
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt attachment staged", result)
}

func (h Handler) Print(c *fiber.Ctx) error {
	result, err := h.service.Print(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	accept := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderAccept)))
	if strings.Contains(accept, fiber.MIMETextHTML) {
		c.Set(fiber.HeaderContentType, result.ContentType)
		return c.Status(fiber.StatusOK).SendString(result.HTML)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt print generated", result)
}

func (h Handler) StockImpact(c *fiber.Ctx) error {
	result, err := h.service.StockImpact(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("id"))
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt stock impact fetched", result)
}

func (h Handler) GenerateDocumentNo(c *fiber.Ctx) error {
	var req GenerateDocumentNoRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.GenerateDocumentNo(c.UserContext(), middleware.ClaimsFromContext(c), req)
	if err != nil {
		return writeReceiptError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse receipt document number generated", result)
}

func writeReceiptError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrReceiptStoreIDRequired), errors.Is(err, ErrReceiptStoreContextRequired), errors.Is(err, ErrReceiptWarehouseRequired), errors.Is(err, ErrReceiptDocumentNoRequired), errors.Is(err, ErrReceiptInvalidVATPercent), errors.Is(err, ErrReceiptItemsRequired), errors.Is(err, ErrReceiptItemNotFound), errors.Is(err, ErrReceiptItemQuantityRequired), errors.Is(err, ErrReceiptItemUnitPriceInvalid), errors.Is(err, ErrReceiptItemLocationRequired), errors.Is(err, ErrReceiptItemLocationMissing), errors.Is(err, ErrReceiptItemProductRequired), errors.Is(err, ErrReceiptDuplicateItem), errors.Is(err, ErrReceiptLocationInactive), errors.Is(err, ErrReceiptLocationWrongStore), errors.Is(err, ErrReceiptLocationWrongWarehouse), errors.Is(err, ErrReceiptLocationSalePoint), errors.Is(err, ErrReceiptProductInactive), errors.Is(err, ErrReceiptAttachmentRequired), errors.Is(err, ErrReceiptAttachmentType), errors.Is(err, ErrReceiptAttachmentSize), errors.Is(err, ErrReceiptAttachmentStorage), errors.Is(err, ErrReceiptConfirmOnlyDraft), errors.Is(err, ErrReceiptConfirmInvalidStatus), errors.Is(err, ErrReceiptSubmitOnlyDraft), errors.Is(err, ErrReceiptReopenInvalidStatus), errors.Is(err, ErrReceiptCancelOnlyDraft), errors.Is(err, ErrReceiptPOQuantityExceeded), errors.Is(err, ErrReceiptImmutable), errors.Is(err, ErrReceiptInvalidStatus), errors.Is(err, ErrInvalidPagination):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrReceiptForbidden), errors.Is(err, ErrReceiptConfirmForbidden), errors.Is(err, ErrReceiptCancelForbidden), errors.Is(err, ErrReceiptReopenForbidden):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrReceiptNotFound), errors.Is(err, ErrReceiptWarehouseNotFound), errors.Is(err, ErrReceiptSupplierNotFound), errors.Is(err, ErrReceiptPurchaseOrderNotFound), errors.Is(err, ErrReceiptProductNotFound), errors.Is(err, ErrReceiptLocationNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	case errors.Is(err, ErrReceiptDuplicateDocumentNo):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	case errors.Is(err, ErrReceiptAttachmentStorageUnavailable):
		return httpx.Error(c, fiber.StatusServiceUnavailable, err.Error(), nil)
	case errors.Is(err, ErrReceiptAttachmentUploadFailed):
		return httpx.Error(c, fiber.StatusBadGateway, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}

func parsePaginationQuery(c *fiber.Ctx) (int, int, error) {
	const defaultPage = 1
	const defaultLimit = 50
	const maxLimit = 200

	page := defaultPage
	limit := defaultLimit
	var err error

	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, err
		}
	}
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, err
		}
	}
	if page < 1 || limit < 1 {
		return 0, 0, ErrInvalidPagination
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit, nil
}

func parseStatusQuery(c *fiber.Ctx) (*ReceiptStatus, error) {
	raw := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if raw == "" {
		return nil, nil
	}
	status := ReceiptStatus(raw)
	switch status {
	case ReceiptStatusDraft, ReceiptStatusPendingReview, ReceiptStatusConfirmed, ReceiptStatusCancelled:
		return &status, nil
	default:
		return nil, ErrReceiptInvalidStatus
	}
}
