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
//  ⚠️ CALIBRATE กับ HTML ที่ render จริงใน DevTools ก่อน production
//     (วาง snippet วัด offsetHeight → ÷ 3.7795 = mm)
//     2 ค่าที่ต้อง lock ให้ตรงกันเสมอ (drift = พังเงียบ):
//       Go rowH (9.5) ↔ CSS --row-h (9.5)
//       Go pageContentH (269) = 297 − padTop(12) − padBottom(16)
const (
	pageContentH = 269.0 // A4 297 − padding บน 12 − padding ล่าง 16
	rowH         = 7.5   // ความสูงต่อแถว — ต้อง lock ให้ตรง CSS --row-h (7.5mm). 1 บรรทัดไทย + clamp 2 บรรทัด

	hHeaderFull = 70.0 // หัวเต็ม (โลโก้/ร้าน/กล่องเลขที่/กล่องลูกค้า+จัดส่ง/ref row) — วัดจริง ~64-70mm
	hHeaderMini = 15.0 // หัวย่อหน้าต่อ
	hTableHead  = 9.0  // thead
	hContinued  = 6.0  // "Continued…"
	hPageNo     = 6.0  // "หน้า X / Y"

	// footer sub-pieces → รวมเป็น footerBlock ตาม profile/data
	hSumRow   = 6.0  // summary 1 บรรทัด
	hSumExtra = 8.0  // ระยะเน้น grand total
	hPayBox   = 48.0 // กล่องชำระเงิน + QR + cheque sub-fields (3 opt rows + bank sub + cheque area ~4 rows)
	hRemarks  = 22.0 // กล่องหมายเหตุ
	hSigBlock = 28.0 // แถวลายเซ็น — ลดลงหลังตัด sig-name ออก (เหลือ ลงชื่อ+line + วันที่)
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
	return geometry{
		contentH:    pageContentH,
		rowH:        rowH,
		headerFull:  hHeaderFull,
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

// paginate = layer แปลง geometry → caps → packRows
func paginate(g geometry, total int) []pageSlice {
	return packRows(
		g.rowsPerPage(g.headerFull, false), // capFirstMid
		g.rowsPerPage(g.headerFull, true),  // capFirstLast (เอกสารหน้าเดียว)
		g.rowsPerPage(g.headerMini, false), // capMid
		g.rowsPerPage(g.headerMini, true),  // capMidLast
		total,
	)
}

// packRows = core algorithm (pure ints) — GREEDY fill (เหมือนฟอร์มตัวอย่าง):
//   เติมสินค้าจริงให้เต็มทุกหน้าก่อน spill ไปหน้าถัดไป → ไม่มีหน้ากลางที่โหรงเหรง
//   (filler เปล่าจะเหลือเฉพาะหน้าสุดท้ายหลังวางสินค้าครบแล้ว)
//   รับประกัน (พิสูจน์ใน pagination_test.go ด้วย property test 0..300 × 5 configs):
//     • รวมแถวทุกหน้า == total, หน้าต่อเนื่องกัน
//     • ไม่มีหน้าเกิน cap ของตัวเอง
//     • IsLast = หน้าสุดท้ายหน้าเดียว, IsFirst = หน้าแรกหน้าเดียว
//     • ไม่มีหน้า summary-only เปล่า (หน้าสุดท้ายมี ≥1 แถวเสมอเมื่อ total≥1)
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
			// สินค้าที่เหลือ + summary ลงหน้านี้ได้ → หน้าสุดท้าย
			pages = append(pages, pageSlice{Start: i, End: total, IsFirst: first, IsLast: true})
			i = total
			continue
		}
		// ยังลง summary หน้านี้ไม่ได้ → อัดสินค้าลงหน้านี้
		// Balance case: ถ้า rem ≤ cap+soloCap (จบใน 2 หน้า) ให้กระจายเท่าๆ กัน
		// แทน greedy (ยัดเต็ม+เหลือนิดเดียว) → กัน "1 item โดดเดี่ยวหน้าสุดท้าย"
		take := cap
		if rem <= cap+soloCap {
			// distribute: give current page ceil(rem/2) but clamp to [rem-soloCap, cap]
			balanced := (rem + 1) / 2
			if balanced < rem-soloCap {
				balanced = rem - soloCap
			}
			if balanced > cap {
				balanced = cap
			}
			take = balanced
		}
		if take >= rem {
			take = rem - 1
		}
		if take < 1 {
			take = 1
		}
		pages = append(pages, pageSlice{Start: i, End: i + take, IsFirst: first, IsLast: false})
		i += take
		first = false
	}
	return pages
}
