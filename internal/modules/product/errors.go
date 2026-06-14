package product

import "errors"

var (
	ErrInvalidProductName      = errors.New("product name is required")
	ErrInvalidMinStock         = errors.New("min stock must be greater than or equal to zero")
	ErrInvalidMaxStock         = errors.New("max stock must be greater than or equal to min stock")
	ErrInvalidBasePrice        = errors.New("base price must be greater than or equal to zero")
	ErrInvalidSpecialPrice     = errors.New("special price must be less than or equal to base price")
	ErrInvalidSpecialPriceDate = errors.New("special price end must be after start")
	ErrInvalidProductTypeID    = errors.New("product type does not belong to this store")
	ErrInvalidProductUnitID    = errors.New("product unit does not belong to this store")
	ErrInvalidBrandID          = errors.New("brand does not belong to this store")
	ErrInvalidDefaultLocation  = errors.New("default storage location does not belong to this store")
	// Phase W2 §4: a product needs a deterministic default location. When none is given
	// and the store has no valid default sale location, creation is rejected with this
	// user-facing Thai message rather than silently leaving the product locationless.
	ErrNoStoreDefaultSaleLocation = errors.New("ไม่พบตำแหน่งขายเริ่มต้นของร้าน กรุณาตั้งค่าคลังสินค้าและตำแหน่งหน้าร้านก่อนสร้างสินค้า")
	ErrInvalidPagination          = errors.New("invalid pagination query")
	ErrInvalidStockStatus         = errors.New("invalid stock_status filter")
	ErrForbiddenStoreAccess       = errors.New("user cannot manage this store")
	ErrProductNotFound            = errors.New("product not found")
	ErrGenerateSKUFailed          = errors.New("unable to generate unique barcode")
	ErrProductInUse               = errors.New("cannot delete product: it is referenced by active purchase orders or other records")
)
