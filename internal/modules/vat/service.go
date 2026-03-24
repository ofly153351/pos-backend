package vat

import "math"

type Service struct{}

func NewService() Service {
	return Service{}
}

func (s Service) Calculate(req CalculateVATRequest) (VATSummary, error) {
	if len(req.Items) == 0 {
		return VATSummary{}, ErrNoItemsProvided
	}

	vatPercent := req.VATPercent
	if vatPercent <= 0 {
		vatPercent = 7
	}

	vatIncluded := true
	if req.VATIncluded != nil {
		vatIncluded = *req.VATIncluded
	}

	subtotal := 0.0
	discountItem := 0.0
	for _, item := range req.Items {
		qty := float64(item.Qty)
		unitPrice := item.Price
		if qty < 0 {
			qty = 0
		}
		if unitPrice < 0 {
			unitPrice = 0
		}
		line := unitPrice * qty
		subtotal += line
		discountItem += item.DiscountPerUnit * qty
	}

	discountBill := req.DiscountBill
	if discountBill < 0 {
		discountBill = 0
	}

	afterDiscount := subtotal - discountItem - discountBill
	if afterDiscount < 0 {
		afterDiscount = 0
	}

	vatAmount := 0.0
	grandTotal := afterDiscount
	if vatPercent > 0 {
		if vatIncluded {
			vatAmount = afterDiscount * vatPercent / (100 + vatPercent)
		} else {
			vatAmount = afterDiscount * vatPercent / 100
			grandTotal = afterDiscount + vatAmount
		}
	}

	return VATSummary{
		Subtotal:      roundMoney(subtotal),
		DiscountItem:  roundMoney(discountItem),
		DiscountBill:  roundMoney(discountBill),
		AfterDiscount: roundMoney(afterDiscount),
		VATPercent:    vatPercent,
		VATAmount:     roundMoney(vatAmount),
		GrandTotal:    roundMoney(grandTotal),
		VATIncluded:   vatIncluded,
	}, nil
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
