package sale

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
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

	// Phase W4B — request idempotency. A retried sale-create carrying the same
	// Idempotency-Key must NOT create a second sale / second payment / second stock
	// deduction. Fingerprint the business intent; if a sale already exists under this
	// key, return it when the intent matches, or reject the conflict when it differs.
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	fingerprint := saleFingerprint(storeID, actor.UserID, req)
	if idempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, storeID, idempotencyKey)
		if err != nil {
			return Sale{}, err
		}
		if existing != nil {
			if existing.RequestFingerprint != fingerprint {
				return Sale{}, ErrSaleIdempotencyConflict
			}
			// Same key + same intent → the original sale, fully hydrated.
			return s.repo.GetByID(ctx, storeID, existing.ID)
		}
	}

	now := time.Now().UTC()
	sale := Sale{
		ID:                 newID(),
		StoreID:            storeID,
		LocationID:         strings.TrimSpace(req.LocationID),
		SaleNumber:         newSaleNumber(now),
		CashierUserID:      actor.UserID,
		Status:             saleStatusCompleted,
		PaymentMethod:      normalizePaymentMethod(req.PaymentMethod),
		PaidAmount:         req.PaidAmount,
		Note:               strings.TrimSpace(req.Note),
		CustomerID:         strings.TrimSpace(req.CustomerID),
		IdempotencyKey:     idempotencyKey,
		RequestFingerprint: fingerprint,
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

	// Owner/manager (or platform_admin) may apply a manual bill discount up to the
	// remaining subtotal; cashiers are capped at 20% (enforced in resolveBillDiscount).
	isElevated, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Sale{}, err
	}

	created, err := s.repo.Create(ctx, sale, DiscountInput{
		ManualDiscount: req.ManualDiscount,
		PromoDiscount:  req.PromoDiscount,
		PromotionIDs:   req.PromotionIDs,
		LegacyBill:     req.DiscountBill,
		IsElevated:     isElevated,
	})
	if err != nil {
		// Concurrent idempotency race: a sibling request carrying the same key committed
		// first (our pre-check ran before it landed). Re-resolve the now-persisted winner;
		// if its intent matches ours, honor the idempotency contract and return THAT sale
		// instead of surfacing a conflict. Only a genuine fingerprint mismatch conflicts.
		if errors.Is(err, ErrSaleIdempotencyConflict) && idempotencyKey != "" {
			existing, ferr := s.repo.FindByIdempotencyKey(ctx, storeID, idempotencyKey)
			if ferr != nil {
				return Sale{}, ferr
			}
			if existing != nil && existing.RequestFingerprint == fingerprint {
				return s.repo.GetByID(ctx, storeID, existing.ID)
			}
		}
		return Sale{}, err
	}
	return created, nil
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

func (s Service) VoidSale(ctx context.Context, actor auth.Claims, storeID, saleID string, req VoidSaleRequest) error {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbiddenStoreAccess
	}
	voidType := strings.TrimSpace(req.Type)
	if voidType == "" {
		voidType = "void"
	}
	if voidType != "void" && voidType != "return" {
		return ErrInvalidVoidType
	}
	return s.repo.VoidSale(ctx, storeID, saleID, actor.UserID, strings.TrimSpace(req.Reason), voidType)
}

// CreateReturn records a partial-or-full return against a sale (owner/manager
// only, mirroring void). The sale is NOT voided; the repository restocks the
// returned quantities and recomputes the sale status.
func (s Service) CreateReturn(ctx context.Context, actor auth.Claims, storeID, saleID string, req CreateReturnRequest) (SaleReturn, error) {
	allowed, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return SaleReturn{}, err
	}
	if !allowed {
		return SaleReturn{}, ErrForbiddenStoreAccess
	}
	if len(req.Items) == 0 {
		return SaleReturn{}, ErrInvalidReturnItems
	}
	return s.repo.CreateReturn(ctx, storeID, saleID, actor.UserID, req)
}

// saleFingerprint is a stable SHA-256 over the business-meaningful fields of a sale
// request (store, sale location, payment, discounts, customer, VAT, promotions, and the
// item set). Two requests with the same fingerprint represent the same sale intent; a
// retry under the same Idempotency-Key with a different fingerprint is a conflict. Item
// tuples and promotion ids are sorted so a re-ordered-but-equivalent cart is treated as
// the same intent. Fields are joined with the Unit/Record separators to avoid collisions.
func saleFingerprint(storeID, actorUserID string, req CreateSaleRequest) string {
	const fieldSep = "\x1f" // unit separator
	const partSep = "\x1e"  // record separator (within an item tuple)
	var b strings.Builder
	write := func(parts ...string) {
		for _, p := range parts {
			b.WriteString(p)
			b.WriteString(fieldSep)
		}
	}
	write(storeID)
	write(strings.TrimSpace(actorUserID))
	write(strings.TrimSpace(req.LocationID))
	write(normalizePaymentMethod(req.PaymentMethod))
	write(strconv.FormatFloat(req.PaidAmount, 'f', 2, 64))
	write(strconv.FormatFloat(req.DiscountBill, 'f', 2, 64))
	write(strings.TrimSpace(req.CustomerID))
	write(strings.TrimSpace(req.Note))
	write(ptrFloatStr(req.ManualDiscount), ptrFloatStr(req.PromoDiscount))
	write(ptrBoolStr(req.VATIncluded), ptrFloatStr(req.VATPercent))

	promoIDs := append([]string(nil), req.PromotionIDs...)
	sort.Strings(promoIDs)
	for _, id := range promoIDs {
		write("promo", strings.TrimSpace(id))
	}

	lines := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		lines = append(lines, strings.Join([]string{
			strings.TrimSpace(it.ProductID),
			strconv.Itoa(it.Quantity),
			normalizeDiscountType(it.DiscountType),
			ptrFloatStr(it.DiscountValue),
		}, partSep))
	}
	sort.Strings(lines)
	for _, l := range lines {
		write("item", l)
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func ptrFloatStr(v *float64) string {
	if v == nil {
		return "nil"
	}
	return strconv.FormatFloat(*v, 'f', 4, 64)
}

func ptrBoolStr(v *bool) string {
	if v == nil {
		return "nil"
	}
	return strconv.FormatBool(*v)
}

func validateCreateRequest(req CreateSaleRequest) error {
	method := normalizePaymentMethod(req.PaymentMethod)
	if method == "" || !AllowedPaymentMethods[method] {
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
