package document

import (
	"errors"
	"testing"
)

func idPtr(s string) *string { return &s }

func TestValidateCreateLines_RejectsInvalidLines(t *testing.T) {
	cases := []struct {
		name    string
		req     CreateDocumentRequest
		wantErr bool
		want    []string // expected indexed fields when wantErr
	}{
		{
			name: "stale empty draft line rejected (no product, no description)",
			req: CreateDocumentRequest{
				Items: []CreateDocumentItemInput{{Description: "", Quantity: 1, UnitPrice: 0}},
			},
			wantErr: true,
			want:    []string{"items[0].description"},
		},
		{
			name: "zero quantity rejected",
			req: CreateDocumentRequest{
				Items: []CreateDocumentItemInput{{Description: "ของฟรี", Quantity: 0, UnitPrice: 10}},
			},
			wantErr: true,
			want:    []string{"items[0].quantity"},
		},
		{
			name: "negative price rejected",
			req: CreateDocumentRequest{
				Items: []CreateDocumentItemInput{{Description: "ส่วนลดลบ", Quantity: 1, UnitPrice: -5}},
			},
			wantErr: true,
			want:    []string{"items[0].unit_price"},
		},
		{
			name: "free-text description line is valid",
			req: CreateDocumentRequest{
				Items: []CreateDocumentItemInput{{Description: "ค่าขนส่งพิเศษ", Quantity: 1, UnitPrice: 50}},
			},
			wantErr: false,
		},
		{
			name: "product line is valid",
			req: CreateDocumentRequest{
				Items: []CreateDocumentItemInput{{ProductID: idPtr("pd-1"), Description: "สินค้า", Quantity: 2, UnitPrice: 100}},
			},
			wantErr: false,
		},
		{
			name: "multiple bad lines report each index",
			req: CreateDocumentRequest{
				Items: []CreateDocumentItemInput{
					{Description: "", Quantity: 1, UnitPrice: 0},          // items[0].description
					{Description: "ok", Quantity: 0, UnitPrice: -1},       // items[1].quantity + unit_price
				},
			},
			wantErr: true,
			want:    []string{"items[0].description", "items[1].quantity", "items[1].unit_price"},
		},
		{
			name: "percent discount out of range rejected",
			req: CreateDocumentRequest{
				Items: []CreateDocumentItemInput{{Description: "x", Quantity: 1, UnitPrice: 100, DiscountType: "PERCENT", DiscountValue: 150}},
			},
			wantErr: true,
			want:    []string{"items[0].discount_value"},
		},
		{
			name: "negative vat rate rejected",
			req: CreateDocumentRequest{
				VatRate: -1,
				Items:   []CreateDocumentItemInput{{Description: "x", Quantity: 1, UnitPrice: 100}},
			},
			wantErr: true,
			want:    []string{"vat_rate"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCreateLines(tc.req)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			var fve *fieldValidationError
			if !errors.As(err, &fve) {
				t.Fatalf("expected fieldValidationError, got %v", err)
			}
			got := map[string]bool{}
			for _, f := range fve.fields {
				got[f.field] = true
			}
			for _, want := range tc.want {
				if !got[want] {
					t.Errorf("missing field %q in %+v", want, fve.fields)
				}
			}
		})
	}
}
