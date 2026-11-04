package sale

import (
	"context"
	"math"
	"testing"
)

func TestPromoCeilingFailsClosedForInvalidCampaigns(t *testing.T) {
	cases := []struct{ typ, data string }{
		{"percentage", `{"percentOff":10,"percentoff":150}`},
		{"fixed_price", `{"fixedPrice":50,"fixedprice":-1}`},
		{"percentage", `{"percentOff":150}`},
		{"happy_hour", `{"percentOff":150}`},
		{"percentage", `{"percentOff":-1}`},
		{"percentage", `{"percentOff":10,"amountOff":"bad"}`},
		{"percentage", `{"percentOff":10`},
		{"percentage", `{"percentOff":null}`},
		{"fixed_price", `{"fixedPrice":-1}`},
		{"fixed_price", `{}`},
		{"bundle", `{"bundlePrice":-1}`},
		{"member_price", `{"memberPrice":-1}`},
		{"buy_x_get_y", `{"buyQty":-1,"getQty":2}`},
	}
	for _, c := range cases {
		t.Run(c.typ+"/"+c.data, func(t *testing.T) {
			if got := promoCeiling(c.typ, c.data, 100, 2); got != 0 {
				t.Errorf("unsafe campaign ceiling=%v, want 0", got)
			}
		})
	}
}

func TestBillDiscountRejectsInvalidSubtotal(t *testing.T) {
	for _, subtotal := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, _, err := (PostgresRepository{}).resolveBillDiscount(context.Background(), nil, "store-a", subtotal, 1, DiscountInput{})
		if err == nil {
			t.Errorf("subtotal=%v accepted", subtotal)
		}
	}
}

func TestCalculateDiscountRejectsInvalidPrice(t *testing.T) {
	for _, price := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := calculateDiscount("", nil, price); err == nil {
			t.Errorf("price=%v accepted", price)
		}
	}
}
