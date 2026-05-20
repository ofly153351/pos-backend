package product

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

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
	req, err := parseCreateRequest(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Create(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), req)
	if err != nil {
		return writeProductError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "product created", result)
}

func (h Handler) ListByStore(c *fiber.Ctx) error {
	page, limit, err := parsePaginationQuery(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, ErrInvalidPagination.Error(), err.Error())
	}
	result, err := h.service.ListByStore(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), ListProductsQuery{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		return writeProductError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "products fetched", result)
}

func (h Handler) GetByID(c *fiber.Ctx) error {
	result, err := h.service.GetByID(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("productID"))
	if err != nil {
		return writeProductError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product fetched", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	req, err := parseUpdateRequest(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.Update(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("productID"), req)
	if err != nil {
		return writeProductError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product updated", result)
}

func (h Handler) Delete(c *fiber.Ctx) error {
	if err := h.service.Delete(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"), c.Params("productID")); err != nil {
		return writeProductError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "product deleted", nil)
}

func (h Handler) GenerateMissingBarcodes(c *fiber.Ctx) error {
	result, err := h.service.GenerateMissingSKU(c.UserContext(), middleware.ClaimsFromContext(c), c.Params("storeID"))
	if err != nil {
		return writeProductError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "missing product barcodes generated", result)
}

func writeProductError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidProductName), errors.Is(err, ErrInvalidQuantity), errors.Is(err, ErrInvalidMinStock), errors.Is(err, ErrInvalidMaxStock), errors.Is(err, ErrInvalidBasePrice), errors.Is(err, ErrInvalidSpecialPrice), errors.Is(err, ErrInvalidSpecialPriceDate), errors.Is(err, ErrInvalidProductTypeID), errors.Is(err, ErrInvalidProductUnitID), errors.Is(err, ErrInvalidBrandID), errors.Is(err, ErrInvalidPagination):
		return httpx.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrForbiddenStoreAccess):
		return httpx.Error(c, fiber.StatusForbidden, err.Error(), nil)
	case errors.Is(err, ErrProductNotFound):
		return httpx.Error(c, fiber.StatusNotFound, err.Error(), nil)
	case errors.Is(err, ErrProductInUse):
		return httpx.Error(c, fiber.StatusConflict, err.Error(), nil)
	default:
		claims := middleware.ClaimsFromContext(c)
		log.Printf(
			"[product] internal error: method=%s path=%s store_id=%s product_id=%s user_id=%s role=%s err=%v",
			c.Method(),
			c.Path(),
			c.Params("storeID"),
			c.Params("productID"),
			claims.UserID,
			claims.Role,
			err,
		)
		return httpx.Error(c, fiber.StatusInternalServerError, "internal server error", nil)
	}
}

func parseCreateRequest(c *fiber.Ctx) (CreateProductRequest, error) {
	req := CreateProductRequest{
		Name:          c.FormValue("name"),
		BrandID:       c.FormValue("brand_id"),
		SKU:           c.FormValue("sku"),
		Barcode:       c.FormValue("barcode"),
		ProductCode:   strings.TrimSpace(c.FormValue("product_code")),
		Description:   strings.TrimSpace(c.FormValue("description")),
		StorageLocation: strings.TrimSpace(c.FormValue("storage_location")),
		ProductTypeID: c.FormValue("product_type_id"),
		ProductUnitID: c.FormValue("unit_id"),
	}
	if value := strings.TrimSpace(c.FormValue("quantity")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return CreateProductRequest{}, err
		}
		req.Quantity = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("min_stock")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return CreateProductRequest{}, err
		}
		req.MinStock = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("max_stock")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return CreateProductRequest{}, err
		}
		req.MaxStock = &parsed
	}
	basePrice, err := parseRequiredFloat(c.FormValue("base_price"))
	if err != nil {
		return CreateProductRequest{}, err
	}
	req.BasePrice = basePrice
	if value := strings.TrimSpace(c.FormValue("cost_price")); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return CreateProductRequest{}, err
		}
		req.CostPrice = parsed
	}
	if value := strings.TrimSpace(c.FormValue("special_price")); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return CreateProductRequest{}, err
		}
		req.SpecialPrice = &parsed
	}
	startAt, endAt, err := parsePriceWindow(c.FormValue("special_price_start_at"), c.FormValue("special_price_end_at"))
	if err != nil {
		return CreateProductRequest{}, err
	}
	req.SpecialPriceStartAt = startAt
	req.SpecialPriceEndAt = endAt
	if value := strings.TrimSpace(c.FormValue("is_active")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return CreateProductRequest{}, err
		}
		req.IsActive = &parsed
	}
	file, err := c.FormFile("image")
	if err == nil {
		req.ImageFile = file
	}
	return req, nil
}

func parseUpdateRequest(c *fiber.Ctx) (UpdateProductRequest, error) {
	req := UpdateProductRequest{}
	if value := c.FormValue("name"); value != "" {
		req.Name = &value
	}
	if value := c.FormValue("brand_id"); value != "" {
		req.BrandID = &value
	}
	if value := c.FormValue("sku"); value != "" {
		req.SKU = &value
	}
	if value := c.FormValue("barcode"); value != "" {
		req.Barcode = &value
	}
	if value := c.FormValue("product_type_id"); value != "" {
		req.ProductTypeID = &value
	}
	if value := c.FormValue("unit_id"); value != "" {
		req.ProductUnitID = &value
	}
	if value := strings.TrimSpace(c.FormValue("quantity")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.Quantity = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("min_stock")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.MinStock = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("max_stock")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.MaxStock = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("base_price")); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.BasePrice = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("cost_price")); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.CostPrice = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("special_price")); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.SpecialPrice = &parsed
	}
	if value := strings.TrimSpace(c.FormValue("clear_special_price")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.ClearSpecialPrice = parsed
	}
	if value := strings.TrimSpace(c.FormValue("clear_max_stock")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.ClearMaxStock = parsed
	}
	if value := strings.TrimSpace(c.FormValue("clear_barcode")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.ClearBarcode = parsed
	}
	if value := strings.TrimSpace(c.FormValue("clear_sku")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.ClearSKU = parsed
	}
	if value := c.FormValue("product_code"); value != "" {
		req.ProductCode = &value
	}
	if value := c.FormValue("description"); value != "" {
		req.Description = &value
	}
	if value := c.FormValue("storage_location"); value != "" {
		req.StorageLocation = &value
	}
	if value := strings.TrimSpace(c.FormValue("clear_product_code")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.ClearProductCode = parsed
	}
	if value := strings.TrimSpace(c.FormValue("clear_special_window")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.ClearSpecialWindow = parsed
	}
	if value := strings.TrimSpace(c.FormValue("is_active")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return UpdateProductRequest{}, err
		}
		req.IsActive = &parsed
	}
	startAt, endAt, err := parsePriceWindow(c.FormValue("special_price_start_at"), c.FormValue("special_price_end_at"))
	if err != nil {
		return UpdateProductRequest{}, err
	}
	req.SpecialPriceStartAt = startAt
	req.SpecialPriceEndAt = endAt
	file, err := c.FormFile("image")
	if err == nil {
		req.ImageFile = file
	}
	return req, nil
}

func parseRequiredFloat(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func parsePriceWindow(startValue, endValue string) (*time.Time, *time.Time, error) {
	var startAt *time.Time
	var endAt *time.Time
	if text := strings.TrimSpace(startValue); text != "" {
		parsed, err := time.Parse(time.RFC3339, text)
		if err != nil {
			return nil, nil, err
		}
		startAt = &parsed
	}
	if text := strings.TrimSpace(endValue); text != "" {
		parsed, err := time.Parse(time.RFC3339, text)
		if err != nil {
			return nil, nil, err
		}
		endAt = &parsed
	}
	return startAt, endAt, nil
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
