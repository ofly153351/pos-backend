package warehouse_receipt

import "errors"

var (
	ErrReceiptStoreIDRequired              = errors.New("store_id is required")
	ErrReceiptStoreContextRequired         = errors.New("store context is required")
	ErrReceiptWarehouseRequired            = errors.New("warehouse_id is required")
	ErrReceiptNotFound                     = errors.New("warehouse receipt not found")
	ErrReceiptForbidden                    = errors.New("user cannot operate this store")
	ErrReceiptConfirmForbidden             = errors.New("user cannot confirm this warehouse receipt")
	ErrReceiptCancelForbidden              = errors.New("user cannot cancel this warehouse receipt")
	ErrReceiptImmutable                    = errors.New("only draft warehouse receipt can be modified")
	ErrReceiptCancelOnlyDraft              = errors.New("only draft warehouse receipt can be cancelled")
	ErrReceiptConfirmOnlyDraft             = errors.New("only draft warehouse receipt can be confirmed")
	ErrReceiptConfirmInvalidStatus         = errors.New("only a draft or pending-review warehouse receipt can be confirmed")
	ErrReceiptSubmitOnlyDraft              = errors.New("only a draft warehouse receipt can be submitted for approval")
	ErrReceiptReopenInvalidStatus          = errors.New("only a warehouse receipt awaiting approval can be reopened to draft")
	ErrReceiptReopenForbidden              = errors.New("user cannot reopen this warehouse receipt")
	ErrReceiptInvalidStatus                = errors.New("invalid warehouse receipt status")
	ErrInvalidPagination                   = errors.New("invalid pagination query")
	ErrReceiptDocumentNoRequired           = errors.New("document_no is required")
	ErrReceiptDuplicateDocumentNo          = errors.New("document_no already exists")
	ErrReceiptInvalidVATPercent            = errors.New("vat_percent must be between 0 and 100")
	ErrReceiptItemsRequired                = errors.New("at least one receipt item is required")
	ErrReceiptItemNotFound                 = errors.New("warehouse receipt item not found")
	ErrReceiptItemQuantityRequired         = errors.New("quantity must be greater than or equal to 1")
	ErrReceiptItemUnitPriceInvalid         = errors.New("unit_price must be greater than or equal to zero")
	ErrReceiptItemLocationRequired         = errors.New("location_id is required")
	ErrReceiptItemLocationMissing          = errors.New("product has no default storage location")
	ErrReceiptItemProductRequired          = errors.New("product_id is required")
	ErrReceiptDuplicateItem                = errors.New("duplicate product and location combination is not allowed")
	ErrReceiptLocationNotFound             = errors.New("location not found")
	ErrReceiptLocationInactive             = errors.New("location is inactive")
	ErrReceiptLocationWrongStore           = errors.New("location does not belong to this store")
	ErrReceiptLocationWrongWarehouse       = errors.New("location does not belong to receipt warehouse")
	// Phase W3 §4: sale-point locations ARE valid receiving destinations
	// (is_sale_point means "ready for sale here", not "cannot receive here"). This
	// error is no longer returned by any resolver; it is retained only so the
	// purchasing module's error mapping continues to compile.
	ErrReceiptLocationSalePoint            = errors.New("sale point locations cannot be used for warehouse receipts")
	ErrReceiptWarehouseNotFound            = errors.New("warehouse not found")
	ErrReceiptSupplierNotFound             = errors.New("supplier not found")
	ErrReceiptPurchaseOrderNotFound        = errors.New("purchase order not found")
	ErrReceiptProductNotFound              = errors.New("product not found")
	ErrReceiptProductInactive              = errors.New("product is inactive")
	ErrReceiptAttachmentRequired           = errors.New("attachment file is required")
	ErrReceiptAttachmentType               = errors.New("attachment file type must be pdf, jpg, or png")
	ErrReceiptAttachmentSize               = errors.New("attachment file size must be less than or equal to 10MB")
	ErrReceiptAttachmentStorage            = errors.New("attachment storage is not configured")
	ErrReceiptAttachmentStorageUnavailable = errors.New("attachment storage is unavailable")
	ErrReceiptAttachmentUploadFailed       = errors.New("failed to upload receipt attachment")
	ErrReceiptPOQuantityExceeded           = errors.New("receipt quantity exceeds outstanding purchase order quantity")
	// Phase W3 — canonical receiving-location + cost + confirm idempotency. User-facing
	// messages are Thai (handler surfaces them directly).
	// §3C: product default is in another warehouse than the receipt.
	ErrReceiptDefaultWrongWarehouse = errors.New("ตำแหน่งเริ่มต้นของสินค้านี้ไม่ได้อยู่ในคลังที่เลือก กรุณาเลือกตำแหน่งรับสินค้าให้ถูกต้อง")
	// §3D: no explicit line location and no valid product default.
	ErrReceiptLocationUnresolved = errors.New("ไม่พบตำแหน่งรับสินค้าที่ถูกต้อง กรุณากำหนดตำแหน่งเริ่มต้นของสินค้าหรือเลือกตำแหน่งรับสินค้า")
	// §9: weighted-average received unit cost computed below zero.
	ErrReceiptCostNegative = errors.New("ต้นทุนรับสินค้าติดลบไม่ได้")
	// §13: confirming an already-confirmed receipt (status guard, user-facing).
	ErrReceiptAlreadyConfirmed = errors.New("ใบรับสินค้านี้ได้รับการยืนยันแล้ว")
	// §12: confirm idempotency key reused with a different request fingerprint.
	ErrReceiptConfirmIdempotencyConflict = errors.New("รหัสคำขอยืนยันนี้ถูกใช้ไปแล้วกับข้อมูลที่ไม่ตรงกัน")
)
