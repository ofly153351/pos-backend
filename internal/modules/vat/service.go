package vat

import (
	"context"
	"math"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/taxcalc"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) CalculateForStore(ctx context.Context, actor auth.Claims, storeID string, req CalculateVATRequest) (VATSummary, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return VATSummary{}, err
	}
	if !allowed {
		return VATSummary{}, ErrForbiddenStoreAccess
	}
	return s.Calculate(req)
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

	// Canonical VAT formula, shared with the sale + invoice modules so a preview can
	// never diverge from what a sale actually charges.
	vatAmount, grandTotal := taxcalc.ComputeVAT(afterDiscount, vatPercent, vatIncluded)

	return VATSummary{
		Subtotal:      roundMoney(subtotal),
		DiscountItem:  roundMoney(discountItem),
		DiscountBill:  roundMoney(discountBill),
		AfterDiscount: roundMoney(afterDiscount),
		VATPercent:    vatPercent,
		VATAmount:     vatAmount,
		GrandTotal:    grandTotal,
		VATIncluded:   vatIncluded,
	}, nil
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
