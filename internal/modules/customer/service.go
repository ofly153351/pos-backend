package customer

import (
	"context"
	"net/mail"
	"pos-backend/internal/platform/fieldcheck"
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

	email, err := normalizeEmail(input.Email)
	if err != nil {
		return Customer{}, err
	}
	if strings.TrimSpace(input.FullName) == "" {
		return Customer{}, ErrInvalidCustomerName
	}
	input.TaxID = fieldcheck.NormalizeTaxID(input.TaxID)
	if err := fieldcheck.First(
		fieldcheck.Phone("phone", input.Phone),
		fieldcheck.Phone("shipping_phone", input.ShippingPhone),
		fieldcheck.TaxID("tax_id", input.TaxID),
	); err != nil {
		return Customer{}, err
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
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, customerID string) (Customer, error) {
	return s.repo.GetByID(ctx, storeID, customerID)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, customerID string, input UpdateCustomerRequest) (Customer, error) {

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

	// B16: validate only changed values; explicit empty clears the field.
	if input.TaxID != nil {
		normalized := fieldcheck.NormalizeTaxID(*input.TaxID)
		input.TaxID = &normalized
	}
	if err := fieldcheck.First(
		fieldcheck.IfChanged("phone", trimmedPtr(input.Phone), existing.Phone, fieldcheck.Phone),
		fieldcheck.IfChanged("shipping_phone", trimmedPtr(input.ShippingPhone), existing.ShippingPhone, fieldcheck.Phone),
		fieldcheck.IfChanged("tax_id", input.TaxID, existing.TaxID, fieldcheck.TaxID),
	); err != nil {
		return Customer{}, err
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
		existing.TaxID = *input.TaxID
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
	return s.repo.Delete(ctx, storeID, customerID)
}

func (s Service) ListLevelDiscounts(ctx context.Context, actor auth.Claims, storeID string) ([]LevelDiscount, error) {
	// POS pricing needs member-tier discounts at operate level so cashiers price member
	// sales identically to owner/manager. Discount writes (Upsert/Delete) remain
	// manage-gated via ensureDiscountAccess.
	return s.repo.ListLevelDiscounts(ctx, storeID)
}

func (s Service) UpsertLevelDiscount(ctx context.Context, actor auth.Claims, storeID string, level int, input UpsertLevelDiscountRequest) (LevelDiscount, error) {
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
	if level <= 0 {
		return ErrInvalidLevel
	}
	return s.repo.DeleteLevelDiscount(ctx, storeID, level)
}

func (s Service) ListShippingAddresses(ctx context.Context, actor auth.Claims, storeID, customerID string) ([]CustomerShippingAddress, error) {
	if _, err := s.repo.GetByID(ctx, storeID, customerID); err != nil {
		return nil, err
	}
	return s.repo.ListShippingAddresses(ctx, customerID)
}

func (s Service) CreateShippingAddress(ctx context.Context, actor auth.Claims, storeID, customerID string, req ShippingAddressRequest) (CustomerShippingAddress, error) {
	if _, err := s.repo.GetByID(ctx, storeID, customerID); err != nil {
		return CustomerShippingAddress{}, err
	}
	if err := fieldcheck.Phone("recipient_phone", req.RecipientPhone); err != nil {
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
	if _, err := s.repo.GetByID(ctx, storeID, customerID); err != nil {
		return err
	}
	return s.repo.DeleteShippingAddress(ctx, addrID, customerID)
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

// trimmedPtr returns a trimmed copy of an optional string (nil stays nil).
func trimmedPtr(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	return &t
}
