package warehouse

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) Create(c *fiber.Ctx) error {
	var req CreateWarehouseRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "warehouse created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"))
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouses fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"))
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	var req UpdateWarehouseRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"), req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	if err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID")); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse deleted", nil)
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidName):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrWarehouseNotFound), errors.Is(err, ErrProductNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	case errors.Is(err, ErrProductNotInWarehouse):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	case errors.Is(err, ErrTransferInvalidDestination), errors.Is(err, ErrTransferSameWarehouse), errors.Is(err, ErrTransferZeroQty):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrInsufficientStock), errors.Is(err, ErrTransferLegacyDisabled):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	case errors.Is(err, ErrInventoryNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	case errors.Is(err, ErrInventoryInsufficientQty), errors.Is(err, ErrAllocateExceedsQty):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	case errors.Is(err, ErrAllocateZeroQty):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrWarehouseInUse), errors.Is(err, ErrProductHasStock), errors.Is(err, ErrWarehouseDirectStockDisabled), errors.Is(err, ErrDefaultWarehouseDelete), errors.Is(err, ErrDefaultWarehouseDeactivate):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	default:
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}

// Warehouse-Product handlers

func (h Handler) AddProduct(c *fiber.Ctx) error {
	var req AddWarehouseProductRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AddProduct(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"), req)
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "product added to warehouse", result)
}

func (h Handler) ListProducts(c *fiber.Ctx) error {
	result, err := h.service.ListProducts(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"))
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse products fetched", result)
}

func (h Handler) UpdateProduct(c *fiber.Ctx) error {
	var req UpdateWarehouseProductRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if req.Quantity == nil || *req.Quantity < 0 {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid quantity", nil)
	}
	if err := h.service.UpdateProduct(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"), c.Params("productID"), *req.Quantity); err != nil {
		return writeError(c, err)
	}
	// Re-fetch the updated product
	result, err := h.service.ListProducts(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"))
	if err != nil {
		return writeError(c, err)
	}
	for _, p := range result {
		if p.ProductID == c.Params("productID") {
			return httpx.Success(c, fiber.StatusOK, "product quantity updated", p)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "product quantity updated", nil)
}

func (h Handler) RemoveProduct(c *fiber.Ctx) error {
	if err := h.service.RemoveProduct(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"), c.Params("productID")); err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product removed from warehouse", nil)
}

// TransferStock handles transferring stock from a warehouse to another warehouse or sale_point.
func (h Handler) TransferStock(c *fiber.Ctx) error {
	var req WarehouseTransferRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.service.TransferStock(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"), req); err != nil {
		log.Printf("TransferStock error: %v", err)
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "stock transferred successfully", nil)
}

// ListInventory lists warehouse inventory items (transferred but not yet allocated).
func (h Handler) ListInventory(c *fiber.Ctx) error {
	result, err := h.service.ListInventory(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"))
	if err != nil {
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "warehouse inventory fetched", result)
}

// AllocateInventory allocates inventory to sellable stock.
func (h Handler) AllocateInventory(c *fiber.Ctx) error {
	var req AllocateInventoryRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.service.AllocateInventory(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("warehouseID"), c.Params("productID"), req); err != nil {
		log.Printf("AllocateInventory error: %v", err)
		return writeError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "inventory allocated successfully", nil)
}
