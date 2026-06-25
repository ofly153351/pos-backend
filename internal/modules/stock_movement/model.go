package stock_movement

import (
	"errors"
	"time"
)

// Movement type constants
const (
	MovementTypeIn              = "IN"
	MovementTypeOut             = "OUT"
	MovementTypeTransfer        = "TRANSFER"
	MovementTypeTransferOut     = "TRANSFER_OUT" // Phase W4A — source leg of a paired transfer
	MovementTypeTransferIn      = "TRANSFER_IN"  // Phase W4A — destination leg of a paired transfer
	MovementTypeSale            = "SALE"
	MovementTypeAdjust          = "ADJUST"
	MovementTypeReturn          = "RETURN"
	MovementTypeCountCorrection = "COUNT_CORRECTION"
	MovementTypeAllocate        = "ALLOCATE" // Phase W5 — warehouse_inventory → stocks allocation
	// NOTE: "OPENING_BALANCE" is a REASON code (see reasons.go opAdd), NOT a movement
	// type. Initial/seed stock is written as type=IN with reason="OPENING_BALANCE", so it
	// is intentionally absent from the type taxonomy.
)

type StockMovement struct {
	ID                    string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID               string    `json:"store_id" gorm:"column:store_id"`
	ProductID             string    `json:"product_id" gorm:"column:product_id"`
	LocationID            *string   `json:"location_id,omitempty" gorm:"column:location_id"`
	DestinationLocationID *string   `json:"destination_location_id,omitempty" gorm:"column:destination_location_id"`
	QuantityChange        int       `json:"quantity_change" gorm:"column:quantity_change"`
	Type                  string    `json:"type" gorm:"column:type"`
	ReferenceID           *string   `json:"reference_id,omitempty" gorm:"column:reference_id"`
	Reason                string    `json:"reason,omitempty" gorm:"column:reason"`
	IdempotencyKey        *string   `json:"-" gorm:"column:idempotency_key"`
	RequestFingerprint    string    `json:"-" gorm:"column:request_fingerprint"`
	Note                  string    `json:"note" gorm:"column:note"`
	CreatedBy             string    `json:"created_by" gorm:"column:created_by"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Relations (read-only)
	ProductName   string `json:"product_name,omitempty"`
	ProductSKU    string `json:"product_sku,omitempty"`
	CreatedByName string `json:"created_by_name,omitempty"`
	LocationName  string `json:"location_name,omitempty"`
}

func (StockMovement) TableName() string {
	return "stock_movements"
}

type AddStockItemRequest struct {
	ProductID      string `json:"product_id"`
	LocationID     string `json:"location_id"`
	Quantity       int    `json:"quantity"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
	Note           string `json:"note"`
}

type AddStockRequest struct {
	Items []AddStockItemRequest `json:"items"`
}

type RemoveStockRequest struct {
	ProductID      string `json:"product_id"`
	LocationID     string `json:"location_id"`
	Quantity       int    `json:"quantity"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
	Note           string `json:"note"`
}

type TransferStockRequest struct {
	ProductID        string `json:"product_id"`
	SourceLocationID string `json:"source_location_id"`
	DestLocationID   string `json:"dest_location_id"`
	Quantity         int    `json:"quantity"`
	Reason           string `json:"reason"`
	Note             string `json:"note"`
	IdempotencyKey   string `json:"idempotency_key"`
}

// StockTransfer is the canonical header/audit row for one location→location transfer
// (Phase W4A). The paired TRANSFER_OUT/TRANSFER_IN movements reference it by id.
type StockTransfer struct {
	ID                 string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID            string    `json:"store_id" gorm:"column:store_id"`
	ProductID          string    `json:"product_id" gorm:"column:product_id"`
	SourceLocationID   string    `json:"source_location_id" gorm:"column:source_location_id"`
	DestLocationID     string    `json:"dest_location_id" gorm:"column:dest_location_id"`
	Quantity           int       `json:"quantity" gorm:"column:quantity"`
	Reason             string    `json:"reason" gorm:"column:reason"`
	Note               string    `json:"note" gorm:"column:note"`
	CreatedBy          string    `json:"created_by" gorm:"column:created_by"`
	IdempotencyKey     *string   `json:"-" gorm:"column:idempotency_key"`
	RequestFingerprint string    `json:"-" gorm:"column:request_fingerprint"`
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at"`
}

func (StockTransfer) TableName() string { return "stock_transfers" }

// TransferResult is the canonical transfer response: the header plus its paired movements.
type TransferResult struct {
	Transfer  StockTransfer   `json:"transfer"`
	Movements []StockMovement `json:"movements"`
}

type AdjustStockRequest struct {
	ProductID   string `json:"product_id"`
	LocationID  string `json:"location_id"`
	PhysicalQty int    `json:"physical_quantity"`
	// ExpectedQty is the caller's view of the CURRENT quantity at LocationID, captured
	// when the count/adjustment was prepared. SET_ACTUAL requires it: inside the locked
	// transaction the service rejects the write unless ExpectedQty == the live location
	// quantity. This is an optimistic lock that (a) makes it impossible to write a
	// store-wide aggregate into one location — a grand total never equals the location's
	// own quantity when stock is split — and (b) detects a stale count (stock changed
	// after counting). A nil value is rejected for SET_ACTUAL (ErrStockExpectedRequired).
	ExpectedQty    *int   `json:"expected_quantity"`
	ReferenceID    string `json:"reference_id"`
	MovementType   string `json:"movement_type"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
	Note           string `json:"note"`
}

type ListMovementsQuery struct {
	ProductID string `json:"product_id"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
}

type MovementResponse struct {
	Items    []StockMovement `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	Limit    int             `json:"limit"`
	LastPage int             `json:"last_page"`
}

type AdditionResult struct {
	Movements []StockMovement `json:"movements"`
}

var (
	ErrStockForbidden        = errors.New("user cannot operate this store")
	ErrStockNoItems          = errors.New("at least one item is required")
	ErrStockBadQty           = errors.New("quantity must be greater than zero")
	ErrProductNotFound       = errors.New("product not found")
	ErrInsufficientStock     = errors.New("insufficient stock quantity")
	ErrLocationMismatch      = errors.New("source and destination locations must be different")
	ErrStockLocationRequired = errors.New("a valid stock location is required")
	// Phase W2 — adjustment reason + result guards. Messages are the user-facing Thai
	// copy (the drawer surfaces backend messages directly).
	ErrStockReasonRequired     = errors.New("กรุณาเลือกเหตุผลในการปรับสต็อก")
	ErrStockReasonInvalid      = errors.New("เหตุผลในการปรับสต็อกไม่ถูกต้องสำหรับการดำเนินการนี้")
	ErrStockReasonNoteRequired = errors.New("กรุณาระบุรายละเอียดเมื่อเลือกเหตุผล \"อื่น ๆ\"")
	ErrStockExceedsAvailable   = errors.New("จำนวนที่ต้องการลดมากกว่าสต็อกคงเหลือในตำแหน่งนี้")
	ErrStockNoChange           = errors.New("ยอดจริงเท่ากับยอดปัจจุบัน ไม่มีการเปลี่ยนแปลงสต็อก")
	// ErrStockExpectedRequired: a SET_ACTUAL adjustment must declare expected_quantity
	// (the location's current on-hand). Without it the server cannot tell a per-location
	// count from a store-wide total, so it refuses rather than risk inflating one location.
	ErrStockExpectedRequired = errors.New("กรุณายืนยันยอดคงเหลือปัจจุบันของตำแหน่งก่อนตั้งยอดจริง")
	// ErrStockStaleCount: expected_quantity does not match the live location quantity. The
	// stock at this location changed after counting, or a store-wide total was sent into a
	// single location. The caller must recount the specific location.
	ErrStockStaleCount = errors.New("ยอดคงเหลือจริงของตำแหน่งนี้ไม่ตรงกับที่นับไว้ (อาจมีสต็อกหลายตำแหน่งหรือมีการเคลื่อนไหวระหว่างนับ) กรุณาตรวจสอบและนับใหม่ตามตำแหน่ง")
	// ErrStockIdempotencyConflict: the same idempotency key was reused with a DIFFERENT
	// request (product/location/quantity/type) — reject rather than return the original.
	ErrStockIdempotencyConflict = errors.New("รหัสคำขอนี้ถูกใช้ไปแล้วกับรายการที่ไม่ตรงกัน")
	// Phase W4A — canonical location-aware transfer guards (Thai user-facing copy).
	ErrTransferSourceRequired      = errors.New("กรุณาเลือกตำแหน่งต้นทาง")
	ErrTransferDestRequired        = errors.New("กรุณาเลือกตำแหน่งปลายทาง")
	ErrTransferSameLocation        = errors.New("ตำแหน่งต้นทางและปลายทางต้องไม่ใช่ตำแหน่งเดียวกัน")
	ErrTransferInsufficient        = errors.New("จำนวนที่ต้องการโอนมากกว่าสต็อกคงเหลือในตำแหน่งต้นทาง")
	ErrTransferLocationNotInStore  = errors.New("ตำแหน่งที่เลือกไม่อยู่ในร้านเดียวกัน")
	ErrTransferLocationInactive    = errors.New("ตำแหน่งที่เลือกไม่สามารถใช้งานได้")
	ErrTransferCrossStore          = errors.New("ยังไม่รองรับการโอนสินค้าข้ามร้าน กรุณาเลือกตำแหน่งภายในร้านเดียวกัน")
	ErrTransferForbidden           = errors.New("คุณไม่มีสิทธิ์โอนย้ายสต็อกสินค้า")
	ErrTransferIdempotencyConflict = errors.New("รหัสคำขอนี้ถูกใช้ไปแล้วกับรายการโอนย้ายที่ไม่ตรงกัน")
)
