package location

import (
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

func parsePage(c *fiber.Ctx) (int, int) {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "0"))
	if page <= 0 {
		page = 1
	}
	if limit < 0 {
		limit = 0
	}
	return page, limit
}

func (h Handler) Create(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req CreateLocationRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNameRequired, ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrLocationExists:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusCreated, "location created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	page, limit := parsePage(c)
	filter := ListFilter{
		WarehouseID: c.Query("warehouse_id"),
		ZoneName:    c.Query("zone_name"),
		FloorName:   c.Query("floor_name"),
		Search:      c.Query("search"),
		Page:        page,
		Limit:       limit,
	}
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), storeID, filter)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "locations fetched", result)
}

func (h Handler) GetTree(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	warehouseID := c.Query("warehouse_id")
	result, err := h.service.ListTree(c.UserContext(), middleware.ClaimsFromContext(c), storeID, warehouseID)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "location tree fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), storeID, locationID)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "location fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	var req UpdateLocationRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), storeID, locationID, req)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrLocationExists:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "location updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), storeID, locationID)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		case ErrLocationInUse:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "location deleted", nil)
}

type renameZoneRequest struct {
	WarehouseID string `json:"warehouse_id"`
	ZoneName    string `json:"zone_name"`
	NewName     string `json:"new_name"`
}

type renameFloorRequest struct {
	WarehouseID string `json:"warehouse_id"`
	ZoneName    string `json:"zone_name"`
	FloorName   string `json:"floor_name"`
	NewName     string `json:"new_name"`
}

func (h Handler) RenameZone(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req renameZoneRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	err := h.service.RenameZone(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req.WarehouseID, req.ZoneName, req.NewName)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "zone renamed", nil)
}

func (h Handler) DeleteZone(c *fiber.Ctx) error {
	storeID    := c.Params("storeID")
	warehouseID := c.Query("warehouse_id")
	zoneName    := c.Query("zone_name")
	err := h.service.DeleteZone(c.UserContext(), middleware.ClaimsFromContext(c), storeID, warehouseID, zoneName)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrLocationInUse:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "zone deleted", nil)
}

func (h Handler) RenameFloor(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req renameFloorRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	err := h.service.RenameFloor(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req.WarehouseID, req.ZoneName, req.FloorName, req.NewName)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "floor renamed", nil)
}

func (h Handler) DeleteFloor(c *fiber.Ctx) error {
	storeID    := c.Params("storeID")
	warehouseID := c.Query("warehouse_id")
	zoneName    := c.Query("zone_name")
	floorName   := c.Query("floor_name")
	err := h.service.DeleteFloor(c.UserContext(), middleware.ClaimsFromContext(c), storeID, warehouseID, zoneName, floorName)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrInvalidWarehouse:
			return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		case ErrLocationInUse:
			return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	return httpx.Success(c, fiber.StatusOK, "floor deleted", nil)
}

func (h Handler) ListProducts(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	locationID := c.Params("locationID")
	page, limit := parsePage(c)
	if limit <= 0 {
		limit = 20
	}
	products, total, err := h.service.ListProducts(c.UserContext(), middleware.ClaimsFromContext(c), storeID, locationID, page, limit)
	if err != nil {
		switch err {
		case ErrLocationForbidden:
			return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
		case ErrLocationNotFound:
			return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
		default:
			return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
		}
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return httpx.Success(c, fiber.StatusOK, "location products fetched", map[string]any{
		"items":       products,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}
