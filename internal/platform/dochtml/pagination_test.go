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
			if total >= 1 {
				if lp := ps[len(ps)-1]; lp.End-lp.Start < 1 {
					t.Fatalf("caps%v total=%d: empty last page (orphan / empty-summary)", c, total)
				}
			}
		}
	}
}

// Boundary table: caps first=15/10 (mid/last), mid=24/18 (mid/last).
// BALANCED: ถ้า rem ≤ cap+soloCap (จบใน 2 หน้า) กระจาย ceil(rem/2) แทน greedy
// → ป้องกัน "1 item โดดเดี่ยวหน้าสุดท้าย" ในเอกสารที่มีสินค้าเยอะ
func TestPackRows_Cases(t *testing.T) {
	cases := []struct {
		total int
		want  [][2]int
	}{
		{0, [][2]int{{0, 0}}},
		{1, [][2]int{{0, 1}}},
		{10, [][2]int{{0, 10}}},           // == capFirstLast → หน้าเดียว
		{11, [][2]int{{0, 6}, {6, 11}}},   // balanced: ceil(11/2)=6 → 6/5 แทน 10/1
		{15, [][2]int{{0, 8}, {8, 15}}},   // balanced: ceil(15/2)=8 → 8/7 แทน 14/1
		{33, [][2]int{{0, 15}, {15, 33}}}, // หน้าแรกเต็ม 15, ที่เหลือ 18 == capMidLast → 15/18
		{34, [][2]int{{0, 15}, {15, 25}, {25, 34}}}, // balanced: 15 + ceil(19/2)=10 → 15/10/9
		{40, [][2]int{{0, 15}, {15, 28}, {28, 40}}}, // balanced: 15 + ceil(25/2)=13 → 15/13/12
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
