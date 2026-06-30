package dochtml

// =============================================================================
//  pagination.go — การแบ่งหน้าเอกสาร (แยกออกจาก document_render_unified.go)
// -----------------------------------------------------------------------------
//  ⚠️ ACTION ก่อนใช้: ลบบล็อกเหล่านี้ออกจาก document_render_unified.go
//     (ย้ายมาอยู่ไฟล์นี้แล้ว เพื่อแยก concern + ให้ test ได้):
//       • const block §1 (pageContentH ... hSigBlock)
//       • type geometry, func (geometry) rowsPerPage, func buildGeometry
//       • type pageSlice, func paginate
//     ส่วน docProfile.footerBlockH ใน render file ยังใช้ค่าคงที่ในไฟล์นี้ได้
//     (package เดียวกัน) — ไม่ต้องแก้
//
//  ดีไซน์: paginate() = compute caps จาก geometry → ส่งต่อ packRows() (pure ints)
//          → packRows test ได้โดยไม่ผูกกับค่า mm (ดู pagination_test.go)
// =============================================================================

// ---- GEOMETRY (mm) : "วัดครั้งเดียวจาก template จริง แล้ว bake" --------------
//
//	⚠️ CALIBRATE กับ HTML ที่ render จริงใน DevTools ก่อน production
//	   (วาง snippet วัด offsetHeight → ÷ 3.7795 = mm)
//	   2 ค่าที่ต้อง lock ให้ตรงกันเสมอ (drift = พังเงียบ):
//	     Go rowH (9.5) ↔ CSS --row-h (9.5)
//	     Go pageContentH (269) = 297 − padTop(12) − padBottom(16)
const (
	pageContentH = 269.0 // A4 297 − padding บน 12 − padding ล่าง 16
	rowH         = 7.5   // ความสูงต่อแถว — ต้อง lock ให้ตรง CSS --row-h (7.5mm). 1 บรรทัดไทย + clamp 2 บรรทัด

	// หัวเอกสารเต็ม — สูงไม่เท่ากันตามชนิดเอกสาร จึงแยกค่า (ไม่งั้นเอกสารที่หัวสั้น
	// จะถูกสำรองที่เกินจริง → เสียไป 1 แถว ทำให้ split หน้าเร็วเกินควร เช่น 15 รายการ
	// ในใบแจ้งหนี้ที่ควรลงหน้าเดียวได้ กลับโดนดันขึ้นหน้า 2)
	hHeaderFull    = 70.0 // DELIVERY_ORDER: มีกล่องที่อยู่จัดส่ง + แถวอ้างอิงจัดส่ง — วัดจริง ~64-70mm
	hHeaderCompact = 63.0 // เอกสารทั่วไป (INVOICE/RECEIPT/TAX_INVOICE/QUOTATION/BILL/CREDIT_NOTE): ไม่มี refrow จัดส่ง (~7mm)
	hHeaderMini    = 15.0 // หัวย่อหน้าต่อ
	hTableHead     = 9.0  // thead
	hContinued     = 6.0  // "Continued…"
	hPageNo        = 6.0  // "หน้า X / Y"

	// footer sub-pieces → รวมเป็น footerBlock ตาม profile/data
	hSumRow   = 6.0  // summary 1 บรรทัด
	hSumExtra = 8.0  // ระยะเน้น grand total
	hPayBox   = 44.0 // กล่องชำระเงิน + QR(16mm) + cheque sub-fields — ย่อลงให้ footer เล็กลง (จุหน้าเดียวได้มากขึ้น)
	hRemarks  = 16.0 // กล่องหมายเหตุ (min-height 9mm) — ย่อลง
	hSigBlock = 31.0 // แถวลายเซ็น — เผื่อ margin-top ช่องลงชื่อให้กรอกได้ไม่แคบ (signatures+sig-t+sig-write+date)
)

type geometry struct {
	contentH, rowH                    float64
	headerFull, headerMini, tableHead float64
	continued, pageNo, footerBlock    float64
}

// rowsPerPage = จำนวนแถวที่ใส่ได้ในหน้า ตาม role และว่าเป็นหน้าสุดท้ายหรือไม่
func (g geometry) rowsPerPage(headerH float64, isLast bool) int {
	reserved := headerH + g.tableHead + g.pageNo
	if isLast {
		reserved += g.footerBlock // หน้าสุดท้ายกันที่ให้ summary + ลายเซ็น
	} else {
		reserved += g.continued
	}
	n := int((g.contentH - reserved) / g.rowH)
	if n < 1 {
		return 1 // กัน loop ค้าง: อย่างน้อย 1 แถว/หน้าเสมอ
	}
	return n
}

func buildGeometry(p docProfile, hasNotes bool) geometry {
	// เอกสารที่ไม่มีกล่อง/แถวที่อยู่จัดส่ง (ทุกชนิดยกเว้น DELIVERY_ORDER) หัวสั้นกว่า
	// จึงสำรองที่หัวน้อยลง เพื่อให้แถวสินค้าได้พื้นที่คืนมา ~1 แถว
	headerFull := hHeaderCompact
	if p.ShowDeliveryBox {
		headerFull = hHeaderFull
	}
	return geometry{
		contentH:    pageContentH,
		rowH:        rowH,
		headerFull:  headerFull,
		headerMini:  hHeaderMini,
		tableHead:   hTableHead,
		continued:   hContinued,
		pageNo:      hPageNo,
		footerBlock: p.footerBlockH(hasNotes),
	}
}

// ---- PAGINATOR --------------------------------------------------------------

type pageSlice struct {
	Start, End      int
	IsFirst, IsLast bool
}

// firstPageRowCap = เพดานจำนวน "รายการสินค้า" บนหน้าแรก (หัวเต็ม).
// ตั้ง 24 = ความจุสูงสุดที่ geometry คำนวณได้ → หน้าแรกเต็มที่สุด ช่องว่างท้ายหน้าน้อยสุด (~5mm).
// ถ้าหัวเอกสารจริงสูงกว่าที่ประเมิน (headerFull=63 ยังไม่ได้ calibrate ใน DevTools) แล้วล้น
// ให้ลดเป็น 22–23 เพื่อเผื่อ margin. เกินเพดาน → split ไปหน้าถัดไปเสมอ.
const firstPageRowCap = 24

// paginate = layer แปลง geometry → caps → packRows
func paginate(g geometry, total int) []pageSlice {
	capFirstMid := g.rowsPerPage(g.headerFull, false)
	if capFirstMid > firstPageRowCap {
		capFirstMid = firstPageRowCap // เพดานหน้าแรก
	}
	return packRows(
		capFirstMid,                        // capFirstMid (≤ firstPageRowCap)
		g.rowsPerPage(g.headerFull, true),  // capFirstLast (เอกสารหน้าเดียว)
		g.rowsPerPage(g.headerMini, false), // capMid
		g.rowsPerPage(g.headerMini, true),  // capMidLast
		total,
	)
}

// packRows = core algorithm (pure ints) — GREEDY fill + footer-only last page:
//
//	เติมสินค้าให้เต็มทุกหน้าต่อเนื่องก่อน (ไม่หารครึ่ง). เมื่อถึงส่วนท้าย:
//	  • ที่เหลือ ≤ soloCap → ลงหน้าสุดท้ายพร้อม footer (หน้าสุดท้ายมีสินค้า)
//	  • soloCap < ที่เหลือ ≤ cap → สินค้าลงหน้าต่อเนื่องนี้ "ครบ" แล้วเปิดหน้าสุดท้าย
//	    ไว้ให้ footer/สรุปยอดอย่างเดียว (footer-only: Start==End) — กันแถวเปล่าค้าง
//	    บนหน้าสินค้า + กันสินค้า 1 ตัวโดดเดี่ยวหน้าถัดไป
//	  • ที่เหลือ > cap → เติมหน้านี้เต็ม cap แล้วไปต่อ
//	รับประกัน (พิสูจน์ใน pagination_test.go ด้วย property test 0..300 × 5 configs):
//	  • รวมแถวทุกหน้า == total, หน้าต่อเนื่องกัน, ไม่มีหน้าเกิน cap
//	  • IsLast = หน้าสุดท้ายหน้าเดียว, IsFirst = หน้าแรกหน้าเดียว
//	  • หน้าสินค้าทุกหน้า (ยกเว้นหน้าสินค้าหน้าสุดท้าย) เต็ม cap เสมอ — ไม่มีแถวเปล่าค้าง
//	  • หน้าสุดท้ายอาจเป็น footer-only (0 แถว) ได้ แต่ต้องตามหลังหน้าที่มีสินค้าเสมอ
func packRows(capFirstMid, capFirstLast, capMid, capMidLast, total int) []pageSlice {
	if total <= 0 {
		return []pageSlice{{Start: 0, End: 0, IsFirst: true, IsLast: true}}
	}
	// safety: cap ต้อง ≥1 (กรณี calibrate footer สูงผิดปกติ)
	if capFirstMid < 1 {
		capFirstMid = 1
	}
	if capFirstLast < 1 {
		capFirstLast = 1
	}
	if capMid < 1 {
		capMid = 1
	}
	if capMidLast < 1 {
		capMidLast = 1
	}

	var pages []pageSlice
	i, first := 0, true
	for i < total {
		cap, soloCap := capMid, capMidLast
		if first {
			cap, soloCap = capFirstMid, capFirstLast
		}
		rem := total - i

		if rem <= soloCap {
			// สินค้าที่เหลือ + footer ลงหน้านี้ได้ → หน้าสุดท้าย (มีสินค้า)
			pages = append(pages, pageSlice{Start: i, End: total, IsFirst: first, IsLast: true})
			break
		}
		if rem <= cap {
			// สินค้าที่เหลือ "ทั้งหมด" ลงหน้าต่อเนื่องนี้ได้ แต่ใส่ footer ด้วยไม่พอ.
			// → วางสินค้าครบบนหน้านี้ (Continued) แล้วเปิดหน้าสุดท้ายไว้ให้ footer/สรุปยอด
			//   อย่างเดียว. วิธีนี้ไม่ทิ้ง "แถวเปล่า" ค้างบนหน้าสินค้า แล้วดันสินค้า 1 ตัว
			//   ไปโดดเดี่ยวหน้าถัดไป (อาการที่ผู้ใช้เจอกับ 23 รายการ → 22 + แถวเปล่า + 1).
			pages = append(pages, pageSlice{Start: i, End: total, IsFirst: first, IsLast: false})
			pages = append(pages, pageSlice{Start: total, End: total, IsFirst: false, IsLast: true})
			break
		}
		// rem > cap → เติมสินค้าหน้านี้ให้เต็ม cap แล้วไปต่อ (greedy)
		pages = append(pages, pageSlice{Start: i, End: i + cap, IsFirst: first, IsLast: false})
		i += cap
		first = false
	}
	return pages
}
