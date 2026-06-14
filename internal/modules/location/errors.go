package location

import "errors"

var (
	ErrLocationForbidden      = errors.New("user cannot manage this store")
	ErrLocationNotFound       = errors.New("location not found")
	ErrLocationNameRequired   = errors.New("location name is required")
	ErrLocationExists         = errors.New("location with this code already exists in this warehouse")
	ErrInvalidWarehouse       = errors.New("warehouse does not belong to this store")
	ErrLocationInUse          = errors.New("cannot delete location: it has stock records")
	// Phase W1 default-protection guards. The messages are the user-facing Thai copy:
	// W1 is backend-only (the warehouse UI/locale files are owned by a concurrent
	// session), so returning the Thai strings here delivers the correct message to the
	// user without touching those files.
	ErrDefaultSaleLocationDelete     = errors.New("ไม่สามารถลบตำแหน่งขายเริ่มต้นได้ กรุณากำหนดตำแหน่งขายเริ่มต้นใหม่ก่อน")
	ErrDefaultSaleLocationDeactivate = errors.New("ไม่สามารถปิดสถานะจุดขายของตำแหน่งเริ่มต้นได้ กรุณาเลือกจุดขายเริ่มต้นใหม่ก่อน")
	// Phase W1 invariant: the default sale location must stay active (is_active=true).
	ErrDefaultSaleLocationDisable = errors.New("ไม่สามารถปิดใช้งานตำแหน่งขายเริ่มต้นได้ กรุณากำหนดตำแหน่งขายเริ่มต้นใหม่ก่อน")
)
