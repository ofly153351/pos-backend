package stockcount

import "errors"

var (
	ErrStoreIDRequired   = errors.New("storeID is required")
	ErrForbidden         = errors.New("user cannot operate this store")
	ErrSessionIDRequired = errors.New("session id is required")
	ErrSessionIDMismatch = errors.New("session id in body does not match the URL")
	ErrSessionNotFound   = errors.New("stock count session not found")

	ErrNoItemsToApply        = errors.New("no items to apply")
	ErrInvalidApplyItem      = errors.New("apply item is invalid")
	ErrSessionAlreadyApplied = errors.New("stock count session already applied")

	// ErrCountLocationRequired: a session must target one active storage location before its
	// count can be applied. Legacy sessions created without a location cannot be applied —
	// their system quantities are store/warehouse aggregates that are unsafe to write into a
	// single location. Thai user-facing copy (the worksheet surfaces backend messages).
	ErrCountLocationRequired = errors.New("รอบตรวจนับนี้ไม่ได้ระบุตำแหน่งจัดเก็บ จึงไม่สามารถปรับยอดสต็อกได้อย่างปลอดภัย กรุณาสร้างรอบตรวจนับใหม่โดยเลือกตำแหน่ง")
	// ErrCountLocationInvalid: the chosen location is not an active location in this store.
	ErrCountLocationInvalid = errors.New("ตำแหน่งที่เลือกสำหรับตรวจนับไม่ถูกต้องหรือไม่สามารถใช้งานได้")
)
