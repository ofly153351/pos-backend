package warehouse

import "errors"

var (
	ErrInvalidName                = errors.New("warehouse name is required")
	ErrWarehouseNotFound          = errors.New("warehouse not found")
	ErrForbiddenStoreAccess       = errors.New("user cannot manage this store")
	ErrProductNotFound            = errors.New("product not found")
	ErrProductNotInWarehouse      = errors.New("product is not assigned to this warehouse")
	ErrTransferInvalidDestination = errors.New("invalid transfer destination type")
	ErrTransferSameWarehouse      = errors.New("source and destination warehouse must be different")
	ErrTransferZeroQty            = errors.New("transfer quantity must be greater than zero")
	ErrInsufficientStock          = errors.New("insufficient stock quantity for transfer")
	// Phase W4A §11: the legacy warehouse-level transfer (auto source selection, cross-store
	// product cloning, auto-created warehouses/locations) is disabled in favour of the
	// canonical location→location transfer (POST /stores/:id/stock-movements/transfer).
	ErrTransferLegacyDisabled   = errors.New("การโอนย้ายรูปแบบเดิมถูกปิดใช้งาน กรุณาใช้การโอนย้ายตามตำแหน่ง")
	ErrInventoryNotFound        = errors.New("warehouse inventory record not found")
	ErrInventoryInsufficientQty = errors.New("insufficient inventory quantity for allocation")
	ErrAllocateZeroQty          = errors.New("allocation quantity must be greater than zero")
	ErrAllocateExceedsQty       = errors.New("allocation quantity exceeds available inventory")
	// Phase W0 safety guards.
	ErrWarehouseInUse               = errors.New("cannot delete warehouse: it still has locations, stock, or related records")
	ErrProductHasStock              = errors.New("cannot remove product from warehouse while stock remains")
	ErrWarehouseDirectStockDisabled = errors.New("direct warehouse stock changes are disabled; use goods receiving or the inventory stock adjustment")
	// Phase W1 default-protection guard. The message is the user-facing Thai copy: W1
	// is backend-only (the warehouse UI/locale files are owned by a concurrent session),
	// so returning the Thai string here guarantees the correct message reaches the user
	// without touching those files.
	ErrDefaultWarehouseDelete = errors.New("ไม่สามารถลบคลังสินค้าเริ่มต้นของร้านได้ กรุณากำหนดคลังเริ่มต้นใหม่ก่อน")
	// Phase W1 invariant: the store's default warehouse must stay active.
	ErrDefaultWarehouseDeactivate = errors.New("ไม่สามารถปิดใช้งานคลังสินค้าเริ่มต้นของร้านได้ กรุณากำหนดคลังเริ่มต้นใหม่ก่อน")

	// Safe-delete / archive lifecycle (migration 047). These carry the user-facing Thai
	// copy for each structured blocker code; the machine code travels in the HTTP error
	// details so the frontend can render an adaptive remediation modal.
	ErrWarehouseHasStock            = errors.New("ไม่สามารถลบคลังสินค้าได้ เนื่องจากยังมีสินค้าคงเหลือในคลัง กรุณาโอนย้ายหรือปรับสต็อกให้เป็นศูนย์ก่อน")
	ErrWarehouseHasBlockedLocations = errors.New("ไม่สามารถลบคลังสินค้าได้ เนื่องจากมีตำแหน่งจัดเก็บที่ยังถูกใช้งานอยู่ กรุณาจัดการตำแหน่งเหล่านั้นก่อน")
	ErrWarehouseHasOpenOperations   = errors.New("ไม่สามารถลบคลังสินค้าได้ เนื่องจากมีรายการที่กำลังดำเนินการอยู่ (รับสินค้า/ตรวจนับ) กรุณาดำเนินการให้เสร็จก่อน")
	// ErrEntityStateChanged: the dependency state changed between the pre-check assessment
	// and the locked delete — the client should re-assess and confirm again.
	ErrEntityStateChanged = errors.New("สถานะข้อมูลมีการเปลี่ยนแปลงระหว่างการลบ กรุณาลองใหม่อีกครั้ง")
)
