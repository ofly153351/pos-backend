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
			// A zero-row final page is an intentional footer-only page when the
			// remaining products do not fit with the footer.
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
		{11, [][2]int{{0, 11}, {11, 11}}},           // all continuation rows, footer-only
		{15, [][2]int{{0, 15}, {15, 15}}},           // all continuation rows, footer-only
		{33, [][2]int{{0, 15}, {15, 33}}},           // final page can carry the footer
		{34, [][2]int{{0, 15}, {15, 34}, {34, 34}}}, // all remainder rows before footer
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

func TestGeometry_FinalProductPageUsesProfileCapacity(t *testing.T) {
	want := map[string]int{
		"INVOICE": 19, "BILL": 19, "TAX_INVOICE": 23,
		"RECEIPT": 24, "CREDIT_NOTE": 24, "DELIVERY_ORDER": 20,
	}
	for docType, expected := range want {
		g := buildGeometry(profileFor(docType), false)
		if got := g.rowsPerPage(g.headerMini, true); got != expected {
			t.Fatalf("%s: final product page cap = %d, want %d", docType, got, expected)
		}
	}
}

func TestGeometry_QuotationKeepsFooterWithFinalRows(t *testing.T) {
	g := buildGeometry(profileFor("QUOTATION"), false)
	pages := paginate(g, 40)
	if len(pages) != 2 || pages[0].End-pages[0].Start != 19 || pages[1].End-pages[1].Start != 21 || !pages[1].IsLast {
		t.Fatalf("QUOTATION 40 รายการควรได้ [19]+[21+footer], got %v", pages)
	}
}

func TestGeometry_AllDocumentProfilesKeepRowsContinuous(t *testing.T) {
	want := map[string][]int{
		"QUOTATION":      {19, 21},
		"INVOICE":        {19, 21, 0},
		"BILL":           {19, 21, 0},
		"TAX_INVOICE":    {19, 21},
		"RECEIPT":        {19, 21},
		"CREDIT_NOTE":    {19, 21},
		"DELIVERY_ORDER": {19, 21, 0},
	}
	for docType, expected := range want {
		pages := paginate(buildGeometry(profileFor(docType), false), 40)
		if len(pages) != len(expected) {
			t.Fatalf("%s: got %v pages, want %v", docType, pages, expected)
		}
		start := 0
		for i, page := range pages {
			rows := page.End - page.Start
			if page.Start != start || rows != expected[i] || (rows == 0 && i != len(pages)-1) {
				t.Fatalf("%s page %d: got [%d,%d] (%d rows), want start=%d rows=%d", docType, i+1, page.Start, page.End, rows, start, expected[i])
			}
			start = page.End
		}
		if start != 40 || !pages[len(pages)-1].IsLast {
			t.Fatalf("%s: pages do not cover all 40 rows: %v", docType, pages)
		}
	}
}

func TestGeometry_InvoiceNoHalving(t *testing.T) {
	g := buildGeometry(profileFor("INVOICE"), false)
	// The final product page owns the footer and must contain products.
	if ps := paginate(g, 18); len(ps) != 2 || ps[0].End != 18 || ps[1].Start != 18 || ps[1].End != 18 {
		t.Fatalf("INVOICE 18 รายการ: ควรได้ [18]+[footer] แต่ได้ %v", ps)
	}
	if ps := paginate(g, 19); len(ps) != 2 || ps[0].End != 19 || ps[1].Start != 19 || ps[1].End != 19 {
		t.Fatalf("INVOICE 19 รายการ: ควรได้ [19]+[footer] แต่ได้ %v", ps)
	}
	if ps := paginate(g, 20); len(ps) != 2 || ps[0].End != 19 || ps[1].Start != 19 || ps[1].End != 20 {
		t.Fatalf("INVOICE 20 รายการ: ควรได้ [19]+[1+footer] แต่ได้ %v", ps)
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
