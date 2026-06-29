package dochtml

import "testing"

// =============================================================================
//  pagination_test.go — verified passing (go1.22)
//  รัน:  go test ./internal/platform/dochtml/ -run PackRows -v
// =============================================================================

func capFor(p pageSlice, k, cFM, cFL, cM, cML int) int {
	switch {
	case k == 0 && p.IsLast:
		return cFL
	case k == 0:
		return cFM
	case p.IsLast:
		return cML
	default:
		return cM
	}
}

// Property test: invariant ต้อง holds ทุก total ข้าม cap config หลายชุด
func TestPackRows_Invariants(t *testing.T) {
	configs := [][4]int{
		{15, 10, 24, 18}, // delivery order-ish
		{20, 15, 30, 25},
		{25, 20, 35, 30}, // invoice-ish
		{5, 3, 8, 6},     // tiny caps (stress)
		{1, 1, 1, 1},     // degenerate
	}
	for _, c := range configs {
		cFM, cFL, cM, cML := c[0], c[1], c[2], c[3]
		for total := 0; total <= 300; total++ {
			ps := packRows(cFM, cFL, cM, cML, total)
			if len(ps) == 0 {
				t.Fatalf("caps%v total=%d: zero pages", c, total)
			}
			sum := 0
			for k, p := range ps {
				if p.Start > p.End {
					t.Fatalf("caps%v total=%d page%d: Start>End", c, total, k)
				}
				if k == 0 && p.Start != 0 {
					t.Fatalf("caps%v total=%d: first page Start!=0", c, total)
				}
				if k > 0 && p.Start != ps[k-1].End {
					t.Fatalf("caps%v total=%d page%d: not contiguous", c, total, k)
				}
				sum += p.End - p.Start
				if rows, cap := p.End-p.Start, capFor(p, k, cFM, cFL, cM, cML); rows > cap {
					t.Fatalf("caps%v total=%d page%d: rows %d > cap %d", c, total, k, rows, cap)
				}
			}
			if sum != total {
				t.Fatalf("caps%v total=%d: covered %d != total", c, total, sum)
			}
			if !ps[len(ps)-1].IsLast {
				t.Fatalf("caps%v total=%d: final page not IsLast", c, total)
			}
			for k := 0; k < len(ps)-1; k++ {
				if ps[k].IsLast {
					t.Fatalf("caps%v total=%d: non-final IsLast at %d", c, total, k)
				}
			}
			for k := 1; k < len(ps); k++ {
				if ps[k].IsFirst {
					t.Fatalf("caps%v total=%d: IsFirst at %d", c, total, k)
				}
			}
			// หน้าสุดท้ายอาจเป็น footer-only (0 แถว) ได้ แต่ต้องตามหลังหน้าที่มีสินค้า
			// (ห้ามเป็นหน้าเดียวเปล่า หรือ footer-only ซ้อน footer-only)
			if total >= 1 {
				if last := ps[len(ps)-1]; last.End-last.Start == 0 {
					if len(ps) < 2 || ps[len(ps)-2].End-ps[len(ps)-2].Start < 1 {
						t.Fatalf("caps%v total=%d: footer-only page ไม่ได้ตามหลังหน้าสินค้า", c, total)
					}
				}
			}
			// GREEDY: หน้าสินค้าทุกหน้า "ยกเว้นหน้าสินค้าหน้าสุดท้าย" ต้องเต็ม cap เป๊ะ
			// (ถ้าไม่เต็ม = หารครึ่ง หรือ ทิ้งแถวเปล่าแล้วดันสินค้าไปหน้าถัดไป)
			lastItems := len(ps) - 1
			for lastItems >= 0 && ps[lastItems].End == ps[lastItems].Start {
				lastItems-- // ข้ามหน้า footer-only
			}
			for k := 0; k < lastItems; k++ {
				capK := cM
				if k == 0 {
					capK = cFM
				}
				if rows := ps[k].End - ps[k].Start; rows != capK {
					t.Fatalf("caps%v total=%d: หน้าสินค้า %d ไม่เต็ม (rows=%d cap=%d) — หารครึ่ง/ทิ้งแถวเปล่า?", c, total, k, rows, capK)
				}
			}
		}
	}
}

// Boundary table: caps first=15/10 (mid/last), mid=24/18 (mid/last).
// GREEDY + footer-only: เติมหน้าต่อเนื่องให้เต็มก่อน. ถ้าสินค้าที่เหลือลงหน้าต่อเนื่อง
// ได้หมดแต่ใส่ footer ไม่พอ → วางครบบนหน้านั้น แล้วเปิดหน้า footer-only (Start==End).
func TestPackRows_Cases(t *testing.T) {
	cases := []struct {
		total int
		want  [][2]int
	}{
		{0, [][2]int{{0, 0}}},
		{1, [][2]int{{0, 1}}},
		{10, [][2]int{{0, 10}}},                     // == capFirstLast → หน้าเดียว
		{11, [][2]int{{0, 11}, {11, 11}}},           // 11 ลงหน้าต่อเนื่องครบ + footer-only
		{15, [][2]int{{0, 15}, {15, 15}}},           // 15 == capFirstMid ครบ + footer-only
		{33, [][2]int{{0, 15}, {15, 33}}},           // 15 เต็ม, เหลือ 18 == capMidLast → หน้าสุดท้ายมีสินค้า
		{34, [][2]int{{0, 15}, {15, 34}, {34, 34}}}, // 15, เหลือ 19 ลงหน้า 2 ครบ + footer-only
		{40, [][2]int{{0, 15}, {15, 39}, {39, 40}}}, // 15, เติมหน้า 2 เต็ม 24, เหลือ 1 พร้อม footer
	}
	for _, c := range cases {
		ps := packRows(15, 10, 24, 18, c.total)
		if len(ps) != len(c.want) {
			t.Fatalf("total=%d: got %d pages want %d (%v)", c.total, len(ps), len(c.want), ps)
		}
		for k := range ps {
			if ps[k].Start != c.want[k][0] || ps[k].End != c.want[k][1] {
				t.Fatalf("total=%d page%d: got [%d,%d] want [%d,%d]",
					c.total, k, ps[k].Start, ps[k].End, c.want[k][0], c.want[k][1])
			}
		}
	}
}

// Geometry sanity: rowsPerPage ต้องคืน ≥1 เสมอ และหน้าสุดท้าย ≤ หน้ากลาง
func TestGeometry_Caps(t *testing.T) {
	g := buildGeometry(profileFor("DELIVERY_ORDER"), true)
	if c := g.rowsPerPage(g.headerFull, false); c < 1 {
		t.Fatalf("capFirstMid < 1: %d", c)
	}
	mid := g.rowsPerPage(g.headerMini, false)
	last := g.rowsPerPage(g.headerMini, true)
	if last > mid {
		t.Fatalf("capMidLast(%d) ควร ≤ capMid(%d) เพราะหน้าสุดท้ายกันที่ให้ footer", last, mid)
	}
}

// Regression: เอกสารทั่วไป (ไม่มี delivery box) หัวสั้นกว่า → 15 รายการต้องลงหน้าเดียว
// ไม่ใช่ split เป็น 8/7 (อาการที่รายงาน). DELIVERY_ORDER หัวสูงกว่าจึงสำรองที่มากกว่า.
func TestGeometry_CompactHeaderSinglePage(t *testing.T) {
	// หมายเหตุ: กล่อง "หมายเหตุ" ขึ้นเสมอ แต่ footer ถูกย่อลงแล้ว (QR/remarks/ลายเซ็นเล็กลง)
	// → ความจุหน้าเดียวเพิ่มเป็น INVOICE/TAX/RECEIPT/BILL=14, QUOTATION=16 → 13 ลงหน้าเดียวได้ทุกชนิด
	for _, dt := range []string{"INVOICE", "TAX_INVOICE", "RECEIPT", "QUOTATION", "BILL"} {
		g := buildGeometry(profileFor(dt), false)
		if got := len(paginate(g, 13)); got != 1 {
			t.Errorf("%s: 13 รายการควรลงหน้าเดียว got %d หน้า (capFirstLast=%d)",
				dt, got, g.rowsPerPage(g.headerFull, true))
		}
	}
	// DELIVERY_ORDER ตั้งใจให้หัวสูงกว่า — compact ต้องไม่ไปลดของมัน
	if del, inv := buildGeometry(profileFor("DELIVERY_ORDER"), false).headerFull,
		buildGeometry(profileFor("INVOICE"), false).headerFull; del <= inv {
		t.Fatalf("headerFull: DELIVERY_ORDER(%.0f) ควร > เอกสารทั่วไป(%.0f)", del, inv)
	}
}

// Regression (อาการที่รายงาน): ใบแจ้งหนี้ที่สินค้าลงหน้าต่อเนื่องได้หมดแต่ใส่ footer
// ไม่พอ ต้องวางสินค้า "ครบ" บนหน้านั้น (ไม่หารครึ่ง, ไม่ทิ้งแถวเปล่าแล้วดัน 1 ตัวไปหน้า
// ถัดไป) แล้วเปิดหน้า footer-only.
func TestGeometry_InvoiceNoHalving(t *testing.T) {
	g := buildGeometry(profileFor("INVOICE"), false)
	capFM := g.rowsPerPage(g.headerFull, false)
	if capFM > firstPageRowCap {
		capFM = firstPageRowCap // หน้าแรกถูกจำกัดเพดานที่ 20
	}
	capM := g.rowsPerPage(g.headerMini, false)

	// หน้าสินค้าทุกหน้า (ยกเว้นหน้าสินค้าหน้าสุดท้าย) ต้องเต็ม cap — ไม่มีแถวเปล่าค้าง
	for total := 16; total <= 120; total++ {
		ps := paginate(g, total)
		lastItems := len(ps) - 1
		for lastItems >= 0 && ps[lastItems].End == ps[lastItems].Start {
			lastItems--
		}
		for k := 0; k < lastItems; k++ {
			capK := capM
			if k == 0 {
				capK = capFM
			}
			if rows := ps[k].End - ps[k].Start; rows != capK {
				t.Fatalf("INVOICE total=%d: หน้าสินค้า %d ไม่เต็ม (rows=%d cap=%d)", total, k, rows, capK)
			}
		}
	}

	// หน้าแรกจุได้สูงสุด 24: 18 รายการ (>15, ≤24) → [18 รายการ]+[footer-only]
	if ps := paginate(g, 18); len(ps) != 2 || ps[0].End != 18 || ps[1].Start != ps[1].End {
		t.Fatalf("INVOICE 18 รายการ: ควรได้ [18]+[footer-only] แต่ได้ %v", ps)
	}
	// 24 รายการ (เต็มเพดานพอดี) → [24]+[footer-only]
	if ps := paginate(g, 24); len(ps) != 2 || ps[0].End != 24 || ps[1].Start != ps[1].End {
		t.Fatalf("INVOICE 24 รายการ: ควรได้ [24]+[footer-only] แต่ได้ %v", ps)
	}
	// เกิน 24 (เช่น 25): หน้าแรกเต็ม 24 → ที่เหลือ 1 ไปหน้าถัดไปพร้อม footer
	if ps := paginate(g, 25); len(ps) != 2 || ps[0].End != 24 || ps[1].Start == ps[1].End {
		t.Fatalf("INVOICE 25 รายการ: ควรได้ [24]+[1 รายการ+footer] แต่ได้ %v", ps)
	}
}

// เอกสารยาวมาก (ผู้ใช้ระบุได้ถึง ~11000 รายการ): greedy ต้อง
//   - ครอบคลุมครบทุกแถว, หน้าต่อเนื่องกัน, ไม่มีหน้าเกิน cap
//   - หน้ากลางเติมเต็ม capMid เสมอ (ไม่หารครึ่ง/ไม่โล่ง)
//   - จำนวนหน้าใกล้เคียงทฤษฎี (ไม่บวมจาก bug) และจบได้ ไม่ loop ค้าง
func TestGeometry_LargeDocument(t *testing.T) {
	g := buildGeometry(profileFor("INVOICE"), false)
	capFM := g.rowsPerPage(g.headerFull, false)
	capM := g.rowsPerPage(g.headerMini, false)
	capML := g.rowsPerPage(g.headerMini, true)

	for _, total := range []int{500, 5000, 11000} {
		ps := paginate(g, total)
		sum, prevEnd := 0, 0
		for k, p := range ps {
			if p.Start != prevEnd {
				t.Fatalf("total=%d page%d: ไม่ต่อเนื่อง (Start=%d want=%d)", total, k, p.Start, prevEnd)
			}
			rows := p.End - p.Start
			cap := capM
			if k == 0 {
				cap = capFM
			}
			if rows > cap {
				t.Fatalf("total=%d page%d: rows %d > cap %d", total, k, rows, cap)
			}
			// หน้ากลาง (ไม่ใช่หน้าแรก/ก่อนสุดท้าย/สุดท้าย) ต้องเต็ม capMid
			if k > 0 && k < len(ps)-2 && rows != capM {
				t.Fatalf("total=%d page%d: หน้ากลางไม่เต็ม (rows=%d cap=%d)", total, k, rows, capM)
			}
			sum += rows
			prevEnd = p.End
		}
		if sum != total {
			t.Fatalf("total=%d: coverage %d != total", total, sum)
		}
		if !ps[len(ps)-1].IsLast {
			t.Fatalf("total=%d: หน้าสุดท้ายไม่ IsLast", total)
		}
		// ขอบเขตจำนวนหน้า: อย่างน้อย ceil(total/capM), อย่างมาก +2 (หน้าแรก/สุดท้ายจุน้อยกว่า)
		lo := (total + capM - 1) / capM
		hi := (total-capFM+capML-1)/capML + 2
		if len(ps) < lo || len(ps) > hi {
			t.Fatalf("total=%d: จำนวนหน้า %d นอกช่วง [%d,%d] — อาจมี bug บวมหน้า", total, len(ps), lo, hi)
		}
	}
}
