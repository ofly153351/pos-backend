package sale

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo         Repository
	resolver     CustomerBenefitResolver
	settingsRepo SettingsRepository
}

type CustomerBenefitResolver interface {
	Resolve(ctx context.Context, storeID, customerID string) (level int, discountPercent float64, err error)
}

func NewService(repo Repository, resolver CustomerBenefitResolver, settingsRepo SettingsRepository) Service {
	return Service{repo: repo, resolver: resolver, settingsRepo: settingsRepo}
}

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateSaleRequest) (Sale, error) {
	if err := validateCreateRequest(req); err != nil {
		return Sale{}, err
	}

	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Sale{}, err
	}
	if !allowed {
		return Sale{}, ErrForbiddenStoreAccess
	}

	now := time.Now().UTC()
	sale := Sale{
		ID:                 newID(),
		StoreID:            storeID,
		SaleNumber:         newSaleNumber(now),
		CashierUserID:      actor.UserID,
		Status:             saleStatusCompleted,
		PaymentMethod:      strings.TrimSpace(req.PaymentMethod),
		PaidAmount:         req.PaidAmount,
		BillDiscountAmount: roundMoney(req.DiscountBill),
		Note:               strings.TrimSpace(req.Note),
		CustomerID:         strings.TrimSpace(req.CustomerID),
		VATIncluded:        true,
		VATPercent:         7,
		SoldAt:             now,
		CreatedAt:          now,
	}
	if req.VATIncluded != nil {
		sale.VATIncluded = *req.VATIncluded
	}
	if req.VATPercent != nil {
		sale.VATPercent = *req.VATPercent
	}

	if sale.CustomerID != "" {
		if s.resolver == nil {
			return Sale{}, ErrCustomerNotFound
		}
		level, discountPercent, err := s.resolver.Resolve(ctx, storeID, sale.CustomerID)
		if err != nil {
			return Sale{}, err
		}
		sale.CustomerLevel = &level
		sale.NetworkDiscountPercent = discountPercent
	}

	for _, item := range req.Items {
		sale.Items = append(sale.Items, SaleItem{
			ProductID:     strings.TrimSpace(item.ProductID),
			Quantity:      item.Quantity,
			DiscountType:  item.DiscountType,
			DiscountValue: item.DiscountValue,
		})
		sale.TotalItems += item.Quantity
	}

	return s.repo.Create(ctx, sale)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]Sale, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, saleID string) (Sale, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Sale{}, err
	}
	if !allowed {
		return Sale{}, ErrForbiddenStoreAccess
	}
	return s.repo.GetByID(ctx, storeID, saleID)
}

func validateCreateRequest(req CreateSaleRequest) error {
	if strings.TrimSpace(req.PaymentMethod) == "" {
		return ErrInvalidPaymentMethod
	}
	if req.PaidAmount < 0 {
		return ErrInvalidPaidAmount
	}
	if req.DiscountBill < 0 {
		return ErrInvalidBillDiscount
	}
	if len(req.Items) == 0 {
		return ErrInvalidSaleItems
	}
	if req.VATPercent != nil && (*req.VATPercent < 0 || *req.VATPercent > 100) {
		return ErrInvalidVATPercent
	}
	for _, item := range req.Items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 {
			return ErrInvalidSaleItem
		}
		discountType := normalizeDiscountType(item.DiscountType)
		if discountType != "" && discountType != DiscountTypeAmount && discountType != DiscountTypePercent {
			return ErrInvalidDiscountType
		}
		if discountType == "" && item.DiscountValue != nil {
			return ErrInvalidDiscountType
		}
		if discountType != "" && item.DiscountValue == nil {
			return ErrDiscountValueRequired
		}
		if item.DiscountValue != nil && *item.DiscountValue < 0 {
			return ErrInvalidDiscountValue
		}
		if discountType == DiscountTypePercent && item.DiscountValue != nil && *item.DiscountValue > 100 {
			return ErrInvalidPercentDiscount
		}
	}
	return nil
}
