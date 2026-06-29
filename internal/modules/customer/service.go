package customer

import (
	"context"
	"net/mail"
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

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, input CreateCustomerRequest) (Customer, error) {
	if strings.TrimSpace(storeID) == "" {
		return Customer{}, ErrCustomerStoreIDRequired
	}
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return Customer{}, err
	}

	email, err := normalizeEmail(input.Email)
	if err != nil {
		return Customer{}, err
	}
	if strings.TrimSpace(input.FullName) == "" {
		return Customer{}, ErrInvalidCustomerName
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	level := 1
	if input.Level != nil {
		level = *input.Level
	}
	if level <= 0 {
		return Customer{}, ErrInvalidLevel
	}

	item := Customer{
		ID:                 newID(),
		StoreID:            storeID,
		Level:              level,
		FullName:           strings.TrimSpace(input.FullName),
		Phone:              strings.TrimSpace(input.Phone),
		Email:              email,
		Address:            strings.TrimSpace(input.Address),
		Note:               strings.TrimSpace(input.Note),
		TaxID:              strings.TrimSpace(input.TaxID),
		Branch:             strings.TrimSpace(input.Branch),
		ShippingContact:    strings.TrimSpace(input.ShippingContact),
		ShippingPhone:      strings.TrimSpace(input.ShippingPhone),
		ShippingAddress:    strings.TrimSpace(input.ShippingAddress),
		ShippingProvince:   strings.TrimSpace(input.ShippingProvince),
		ShippingDistrict:   strings.TrimSpace(input.ShippingDistrict),
		ShippingPostalCode: strings.TrimSpace(input.ShippingPostalCode),
		DeliveryNote:       strings.TrimSpace(input.DeliveryNote),
		IsActive:           isActive,
		CreatedAt:          time.Now().UTC(),
	}

	return s.repo.Create(ctx, item)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]CustomerListItem, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, customerID string) (Customer, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return Customer{}, err
	}
	return s.repo.GetByID(ctx, storeID, customerID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, customerID string, input UpdateCustomerRequest) (Customer, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return Customer{}, err
	}

	existing, err := s.repo.GetByID(ctx, storeID, customerID)
	if err != nil {
		return Customer{}, err
	}

	if input.FullName != nil {
		existing.FullName = strings.TrimSpace(*input.FullName)
	}
	if strings.TrimSpace(existing.FullName) == "" {
		return Customer{}, ErrInvalidCustomerName
	}

	if input.Level != nil {
		existing.Level = *input.Level
	}
	if existing.Level <= 0 {
		return Customer{}, ErrInvalidLevel
	}

	if input.Phone != nil {
		existing.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Email != nil {
		email, err := normalizeEmail(*input.Email)
		if err != nil {
			return Customer{}, err
		}
		existing.Email = email
	}
	if input.Address != nil {
		existing.Address = strings.TrimSpace(*input.Address)
	}
	if input.Note != nil {
		existing.Note = strings.TrimSpace(*input.Note)
	}
	if input.TaxID != nil {
		existing.TaxID = strings.TrimSpace(*input.TaxID)
	}
	if input.Branch != nil {
		existing.Branch = strings.TrimSpace(*input.Branch)
	}
	if input.ShippingContact != nil {
		existing.ShippingContact = strings.TrimSpace(*input.ShippingContact)
	}
	if input.ShippingPhone != nil {
		existing.ShippingPhone = strings.TrimSpace(*input.ShippingPhone)
	}
	if input.ShippingAddress != nil {
		existing.ShippingAddress = strings.TrimSpace(*input.ShippingAddress)
	}
	if input.ShippingProvince != nil {
		existing.ShippingProvince = strings.TrimSpace(*input.ShippingProvince)
	}
	if input.ShippingDistrict != nil {
		existing.ShippingDistrict = strings.TrimSpace(*input.ShippingDistrict)
	}
	if input.ShippingPostalCode != nil {
		existing.ShippingPostalCode = strings.TrimSpace(*input.ShippingPostalCode)
	}
	if input.DeliveryNote != nil {
		existing.DeliveryNote = strings.TrimSpace(*input.DeliveryNote)
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}

	existing.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, existing)
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, customerID string) error {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, storeID, customerID)
}

func (s Service) ListLevelDiscounts(ctx context.Context, actor auth.Claims, storeID string) ([]LevelDiscount, error) {
	// POS pricing needs member-tier discounts at operate level so cashiers price member
	// sales identically to owner/manager. Discount writes (Upsert/Delete) remain
	// manage-gated via ensureDiscountAccess.
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	return s.repo.ListLevelDiscounts(ctx, storeID)
}

func (s Service) UpsertLevelDiscount(ctx context.Context, actor auth.Claims, storeID string, level int, input UpsertLevelDiscountRequest) (LevelDiscount, error) {
	if err := s.ensureDiscountAccess(ctx, actor, storeID); err != nil {
		return LevelDiscount{}, err
	}
	if level <= 0 {
		return LevelDiscount{}, ErrInvalidLevel
	}
	if input.DiscountPercent < 0 || input.DiscountPercent > 100 {
		return LevelDiscount{}, ErrInvalidDiscountPercent
	}

	item := LevelDiscount{
		StoreID:         storeID,
		Level:           level,
		DiscountPercent: input.DiscountPercent,
		CreatedAt:       time.Now().UTC(),
	}
	return s.repo.UpsertLevelDiscount(ctx, item)
}

func (s Service) DeleteLevelDiscount(ctx context.Context, actor auth.Claims, storeID string, level int) error {
	if err := s.ensureDiscountAccess(ctx, actor, storeID); err != nil {
		return err
	}
	if level <= 0 {
		return ErrInvalidLevel
	}
	return s.repo.DeleteLevelDiscount(ctx, storeID, level)
}

func (s Service) ListShippingAddresses(ctx context.Context, actor auth.Claims, storeID, customerID string) ([]CustomerShippingAddress, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, storeID, customerID); err != nil {
		return nil, err
	}
	return s.repo.ListShippingAddresses(ctx, customerID)
}

func (s Service) CreateShippingAddress(ctx context.Context, actor auth.Claims, storeID, customerID string, req ShippingAddressRequest) (CustomerShippingAddress, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return CustomerShippingAddress{}, err
	}
	if _, err := s.repo.GetByID(ctx, storeID, customerID); err != nil {
		return CustomerShippingAddress{}, err
	}
	now := time.Now().UTC()
	addr := CustomerShippingAddress{
		ID:                 newShippingAddressID(),
		CustomerID:         customerID,
		Label:              strings.TrimSpace(req.Label),
		RecipientName:      strings.TrimSpace(req.RecipientName),
		RecipientPhone:     strings.TrimSpace(req.RecipientPhone),
		Address:            strings.TrimSpace(req.Address),
		SubDistrict:        strings.TrimSpace(req.SubDistrict),
		District:           strings.TrimSpace(req.District),
		Province:           strings.TrimSpace(req.Province),
		PostalCode:         strings.TrimSpace(req.PostalCode),
		Note:               strings.TrimSpace(req.Note),
		UseCustomerAddress: req.UseCustomerAddress,
		IsDefault:          req.IsDefault,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return s.repo.CreateShippingAddress(ctx, addr)
}

func (s Service) UpdateShippingAddress(ctx context.Context, actor auth.Claims, storeID, customerID, addrID string, req ShippingAddressRequest) (CustomerShippingAddress, error) {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return CustomerShippingAddress{}, err
	}
	if _, err := s.repo.GetByID(ctx, storeID, customerID); err != nil {
		return CustomerShippingAddress{}, err
	}
	addr := CustomerShippingAddress{
		ID:                 addrID,
		CustomerID:         customerID,
		Label:              strings.TrimSpace(req.Label),
		RecipientName:      strings.TrimSpace(req.RecipientName),
		RecipientPhone:     strings.TrimSpace(req.RecipientPhone),
		Address:            strings.TrimSpace(req.Address),
		SubDistrict:        strings.TrimSpace(req.SubDistrict),
		District:           strings.TrimSpace(req.District),
		Province:           strings.TrimSpace(req.Province),
		PostalCode:         strings.TrimSpace(req.PostalCode),
		Note:               strings.TrimSpace(req.Note),
		UseCustomerAddress: req.UseCustomerAddress,
		IsDefault:          req.IsDefault,
		UpdatedAt:          time.Now().UTC(),
	}
	return s.repo.UpdateShippingAddress(ctx, addr)
}

func (s Service) DeleteShippingAddress(ctx context.Context, actor auth.Claims, storeID, customerID, addrID string) error {
	if err := s.ensureStoreAccess(ctx, actor, storeID); err != nil {
		return err
	}
	if _, err := s.repo.GetByID(ctx, storeID, customerID); err != nil {
		return err
	}
	return s.repo.DeleteShippingAddress(ctx, addrID, customerID)
}

func (s Service) ensureStoreAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	ok, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCustomerForbidden
	}
	return nil
}

func (s Service) ensureDiscountAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if actor.Role == auth.RolePlatformAdmin {
		return nil
	}
	if actor.Role != auth.RoleOwner && actor.Role != auth.RoleManager {
		return ErrCustomerForbidden
	}
	return s.ensureStoreAccess(ctx, actor, storeID)
}

func normalizeEmail(email string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(email))
	if value == "" {
		return "", nil
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return "", ErrInvalidCustomerEmail
	}
	return value, nil
}

// ValidateCustomerInput exposes the core validation rules for testing without DB access.
func (s Service) ValidateCustomerInput(fullName, email string, level int) error {
	if strings.TrimSpace(fullName) == "" {
		return ErrInvalidCustomerName
	}
	if _, err := normalizeEmail(email); err != nil {
		return err
	}
	if level <= 0 {
		return ErrInvalidLevel
	}
	return nil
}

type SaleBenefitResolver struct {
	repo Repository
}

func NewSaleBenefitResolver(repo Repository) SaleBenefitResolver {
	return SaleBenefitResolver{repo: repo}
}

func (r SaleBenefitResolver) Resolve(ctx context.Context, storeID, customerID string) (int, float64, error) {
	level, err := r.repo.GetNetworkLevel(ctx, storeID, customerID)
	if err != nil {
		return 0, 0, err
	}
	percent, err := r.repo.GetLevelDiscountPercent(ctx, storeID, level)
	if err != nil {
		return 0, 0, err
	}
	return level, percent, nil
}
