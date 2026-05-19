package purchasing

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

// ---- Suppliers ----

func (h Handler) CreateSupplier(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateSupplierRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.CreateSupplier(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "supplier created", result)
}

func (h Handler) ListSuppliers(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.ListSuppliers(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "suppliers fetched", result)
}

func (h Handler) GetSupplier(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	result, err := h.service.GetSupplier(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "supplier fetched", result)
}

func (h Handler) UpdateSupplier(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	var req UpdateSupplierRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.UpdateSupplier(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "supplier updated", result)
}

func (h Handler) DeleteSupplier(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	if err := h.service.DeleteSupplier(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID); err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "supplier deleted", nil)
}

// ---- Purchase Orders ----

func (h Handler) CreatePO(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreatePORequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.CreatePO(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "purchase order created", result)
}

func (h Handler) ListPOs(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.ListPOs(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "purchase orders fetched", result)
}

func (h Handler) GetPO(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	poID := c.Params("poID")
	result, err := h.service.GetPO(c.UserContext(), middleware.ClaimsFromContext(c), storeID, poID)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "purchase order fetched", result)
}

func (h Handler) UpdatePO(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	poID := c.Params("poID")
	var req UpdatePORequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.UpdatePO(c.UserContext(), middleware.ClaimsFromContext(c), storeID, poID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "purchase order updated", result)
}

func (h Handler) ReceiveStock(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	poID := c.Params("poID")
	var req ReceivePORequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.ReceiveStock(c.UserContext(), middleware.ClaimsFromContext(c), storeID, poID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "stock received", result)
}

func (h Handler) CancelPO(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	poID := c.Params("poID")
	result, err := h.service.CancelPO(c.UserContext(), middleware.ClaimsFromContext(c), storeID, poID)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "purchase order cancelled", result)
}

// ---- Supplier Products ----

func (h Handler) ListSupplierProducts(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	result, err := h.service.ListSupplierProducts(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "supplier products fetched", result)
}

func (h Handler) AddSupplierProduct(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	var req AddSupplierProductRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AddSupplierProduct(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "supplier product added", result)
}

func (h Handler) UpdateSupplierProduct(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	productID := c.Params("productID")
	var req UpdateSupplierProductRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.UpdateSupplierProduct(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID, productID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "supplier product updated", result)
}

func (h Handler) RemoveSupplierProduct(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	productID := c.Params("productID")
	if err := h.service.RemoveSupplierProduct(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID, productID); err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "supplier product removed", nil)
}

func (h Handler) CreateSupplierProductWithNewProduct(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	supplierID := c.Params("supplierID")
	var req CreateSupplierProductAndLinkRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.CreateSupplierProductAndLink(c.UserContext(), middleware.ClaimsFromContext(c), storeID, supplierID, req)
	if err != nil {
		return writePurchasingError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "supplier product created", result)
}

func writePurchasingError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrSupplierNameRequired),
		errors.Is(err, ErrSupplierStoreIDReq),
		errors.Is(err, ErrPOStoreIDRequired),
		errors.Is(err, ErrPOInvalidStatus),
		errors.Is(err, ErrPOItemsRequired),
		errors.Is(err, ErrPOInvalidQuantity),
		errors.Is(err, ErrPOInvalidUnitCost),
		errors.Is(err, ErrPOReceiveInvalidQty),
		errors.Is(err, ErrSupplierProductRequired),
		errors.Is(err, ErrSupplierProductNameReq):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrSupplierForbidden),
		errors.Is(err, ErrPOForbidden):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrSupplierNotFound),
		errors.Is(err, ErrPONotFound),
		errors.Is(err, ErrPOProductNotFound),
		errors.Is(err, ErrSupplierProductNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	case errors.Is(err, ErrPOAlreadyCompleted),
		errors.Is(err, ErrPOAlreadyCancelled),
		errors.Is(err, ErrSupplierProductExists):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}
