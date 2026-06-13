package purchasing

import (
	"errors"
	"log"
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

// ---- Suppliers ----

func (h Handler) CreateSupplier(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	req := parseCreateSupplierRequest(c)
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
	req := parseUpdateSupplierRequest(c)
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
	case errors.Is(err, ErrSupplierNameRequired):
		return httpx.Err422(c, "name", err.Error())
	case errors.Is(err, ErrSupplierProductNameReq):
		return httpx.Err422(c, "product_name", err.Error())
	case errors.Is(err, ErrSupplierProductRequired):
		return httpx.Err422(c, "product_id", err.Error())
	case errors.Is(err, ErrPOItemsRequired):
		return httpx.Err422(c, "items", err.Error())
	case errors.Is(err, ErrPOInvalidQuantity), errors.Is(err, ErrPOReceiveInvalidQty):
		return httpx.Err422(c, "quantity", err.Error())
	case errors.Is(err, ErrPOInvalidUnitCost):
		return httpx.Err422(c, "unit_cost", err.Error())
	case errors.Is(err, ErrPOInvalidStatus), errors.Is(err, ErrSupplierStoreIDReq), errors.Is(err, ErrPOStoreIDRequired):
		return httpx.ErrBadRequest(c, err.Error())
	case errors.Is(err, ErrSupplierForbidden), errors.Is(err, ErrPOForbidden):
		return httpx.ErrForbidden(c, err.Error())
	case errors.Is(err, ErrSupplierNotFound), errors.Is(err, ErrPONotFound),
		errors.Is(err, ErrPOProductNotFound), errors.Is(err, ErrSupplierProductNotFound):
		return httpx.ErrNotFound(c, err.Error())
	case errors.Is(err, ErrPOAlreadyCompleted), errors.Is(err, ErrPOAlreadyCancelled),
		errors.Is(err, ErrSupplierProductExists), errors.Is(err, ErrPOOrderNumberConflict):
		return httpx.ErrConflict(c, err.Error())
	default:
		log.Printf("[purchasing] internal error: %v", err)
		return httpx.ErrInternal(c)
	}
}

// ── Form parsers ──────────────────────────────────────────────────────────────

func parseCreateSupplierRequest(c *fiber.Ctx) CreateSupplierRequest {
	creditDays, _ := strconv.Atoi(c.FormValue("credit_days"))
	isActiveStr := c.FormValue("is_active")
	var isActive *bool
	if isActiveStr != "" {
		v := isActiveStr == "true" || isActiveStr == "1"
		isActive = &v
	}
	req := CreateSupplierRequest{
		Name:              c.FormValue("name"),
		Phone:             c.FormValue("phone"),
		Address:           c.FormValue("address"),
		TaxID:             c.FormValue("tax_id"),
		ContactPerson:     c.FormValue("contact_person"),
		Note:              c.FormValue("note"),
		IsActive:          isActive,
		Email:             c.FormValue("email"),
		LineID:            c.FormValue("line_id"),
		PaymentMethod:     c.FormValue("payment_method"),
		PromptpayNumber:   c.FormValue("promptpay_number"),
		BankName:          c.FormValue("bank_name"),
		BankAccountNumber: c.FormValue("bank_account_number"),
		BankAccountName:   c.FormValue("bank_account_name"),
		CreditDays:        creditDays,
	}
	if logo, err := c.FormFile("logo"); err == nil {
		req.LogoFile = logo
	}
	return req
}

func parseUpdateSupplierRequest(c *fiber.Ctx) UpdateSupplierRequest {
	req := UpdateSupplierRequest{}
	if v := c.FormValue("name"); v != "" {
		req.Name = &v
	}
	if v := c.FormValue("phone"); v != "" {
		req.Phone = &v
	}
	if v := c.FormValue("address"); v != "" {
		req.Address = &v
	}
	if v := c.FormValue("tax_id"); v != "" {
		req.TaxID = &v
	}
	if v := c.FormValue("contact_person"); v != "" {
		req.ContactPerson = &v
	}
	if v := c.FormValue("note"); v != "" {
		req.Note = &v
	}
	if v := c.FormValue("is_active"); v != "" {
		b := v == "true" || v == "1"
		req.IsActive = &b
	}
	if v := c.FormValue("email"); v != "" {
		req.Email = &v
	}
	if v := c.FormValue("line_id"); v != "" {
		req.LineID = &v
	}
	if v := c.FormValue("payment_method"); v != "" {
		req.PaymentMethod = &v
	}
	if v := c.FormValue("promptpay_number"); v != "" {
		req.PromptpayNumber = &v
	}
	if v := c.FormValue("bank_name"); v != "" {
		req.BankName = &v
	}
	if v := c.FormValue("bank_account_number"); v != "" {
		req.BankAccountNumber = &v
	}
	if v := c.FormValue("bank_account_name"); v != "" {
		req.BankAccountName = &v
	}
	if v := c.FormValue("credit_days"); v != "" {
		n, _ := strconv.Atoi(v)
		req.CreditDays = &n
	}
	if strings.EqualFold(c.FormValue("remove_logo"), "true") {
		req.RemoveLogo = true
	}
	if logo, err := c.FormFile("logo"); err == nil {
		req.LogoFile = logo
	}
	return req
}
