package parkedbill

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateParkedBillRequest) (ParkedBillResponse, error) {
	if strings.TrimSpace(req.Label) == "" {
		return ParkedBillResponse{}, ErrInvalidLabel
	}
	if req.BillDiscountAmount < 0 {
		return ParkedBillResponse{}, ErrInvalidBillDiscount
	}
	if req.VATPercent != nil && (*req.VATPercent < 0 || *req.VATPercent > 100) {
		return ParkedBillResponse{}, ErrInvalidVATPercent
	}
	if len(req.Items) == 0 {
		return ParkedBillResponse{}, ErrInvalidItems
	}
	for _, item := range req.Items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 {
			return ParkedBillResponse{}, ErrInvalidItem
		}
	}

	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ParkedBillResponse{}, err
	}
	if !allowed {
		return ParkedBillResponse{}, ErrForbiddenStoreAccess
	}

	now := time.Now().UTC()
	bill := ParkedBill{
		ID:                     newID(),
		StoreID:                storeID,
		CashierUserID:          actor.UserID,
		Label:                  strings.TrimSpace(req.Label),
		Note:                   strings.TrimSpace(req.Note),
		BillDiscountAmount:     req.BillDiscountAmount,
		BillDiscountType:       req.BillDiscountType,
		BillDiscountPercent:    req.BillDiscountPercent,
		CustomerID:             strings.TrimSpace(req.CustomerID),
		CustomerSettlementMode: req.CustomerSettlementMode,
		PaymentMethod:          req.PaymentMethod,
		VATIncluded:            true,
		VATPercent:             7,
		CreatedAt:              now,
	}
	if req.VATIncluded != nil {
		bill.VATIncluded = *req.VATIncluded
	}
	if req.VATPercent != nil {
		bill.VATPercent = *req.VATPercent
	}
	if bill.CustomerSettlementMode == "" {
		bill.CustomerSettlementMode = "cash_now"
	}
	if bill.PaymentMethod == "" {
		bill.PaymentMethod = "cash"
	}
	if bill.BillDiscountType == "" {
		bill.BillDiscountType = "amount"
	}

	var items []ParkedBillItem
	for _, item := range req.Items {
		product, err := s.repo.GetProductByID(ctx, strings.TrimSpace(item.ProductID))
		if err != nil {
			return ParkedBillResponse{}, ErrInvalidItem
		}
		items = append(items, ParkedBillItem{
			ID:            newItemID(),
			ParkedBillID:  bill.ID,
			ProductID:     product.ID,
			ProductName:   product.Name,
			ProductSKU:    product.SKU,
			Price:         product.BasePrice,
			Quantity:      item.Quantity,
			DiscountType:  item.DiscountType,
			DiscountValue: item.DiscountValue,
		})
	}
	bill.Items = items

	if err := s.repo.CreateParkedBill(ctx, &bill); err != nil {
		return ParkedBillResponse{}, err
	}
	if err := s.repo.CreateParkedBillItems(ctx, items); err != nil {
		return ParkedBillResponse{}, err
	}

	return ParkedBillResponse{Bill: bill, Items: items}, nil
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]ParkedBill, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, billID string) (ParkedBillResponse, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return ParkedBillResponse{}, err
	}
	if !allowed {
		return ParkedBillResponse{}, ErrForbiddenStoreAccess
	}

	bill, err := s.repo.GetByID(ctx, billID)
	if err != nil {
		return ParkedBillResponse{}, err
	}
	if bill.StoreID != storeID {
		return ParkedBillResponse{}, ErrParkedBillNotFound
	}

	items, err := s.repo.GetItemsByBillID(ctx, billID)
	if err != nil {
		return ParkedBillResponse{}, err
	}
	return ParkedBillResponse{Bill: bill, Items: items}, nil
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, billID string) error {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}

	bill, err := s.repo.GetByID(ctx, billID)
	if err != nil {
		return err
	}
	if bill.StoreID != storeID {
		return ErrParkedBillNotFound
	}

	return s.repo.Delete(ctx, billID)
}
