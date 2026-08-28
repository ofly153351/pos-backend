package store

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"pos-backend/internal/idgen"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/activitycapture"
)

type Service struct {
	repo    Repository
	storage LogoStorage
}

func NewService(repo Repository, storage LogoStorage) Service {
	return Service{
		repo:    repo,
		storage: storage,
	}
}

func (s Service) CreateStore(ctx context.Context, actor auth.Claims, input CreateStoreRequest) (Store, error) {
	if strings.TrimSpace(input.Name) == "" {
		return Store{}, ErrInvalidStoreName
	}

	if strings.TrimSpace(input.CurrencyCode) == "" {
		input.CurrencyCode = "THB"
	}

	if strings.TrimSpace(input.SubscriptionPlanCode) == "" {
		return Store{}, ErrInvalidSubscriptionPlan
	}

	logoURL, err := s.storage.SaveStoreLogo(input.LogoFile)
	if err != nil {
		return Store{}, err
	}

	storeModel := Store{
		ID:           newHexID(),
		OwnerUserID:  actor.UserID,
		Name:         strings.TrimSpace(input.Name),
		LogoURL:      logoURL,
		Phone:        strings.TrimSpace(input.Phone),
		Address:      strings.TrimSpace(input.Address),
		PromptPayID:  strings.TrimSpace(input.PromptPayID),
		CurrencyCode: strings.ToUpper(strings.TrimSpace(input.CurrencyCode)),
		CreatedAt:    time.Now().UTC(),
	}

	created, err := s.repo.CreateWithOwner(ctx, storeModel, actor.UserID, strings.TrimSpace(input.SubscriptionPlanCode))
	if err != nil {
		if logoURL != "" {
			if deleteErr := s.storage.DeleteStoreLogo(logoURL); deleteErr != nil {
				slog.WarnContext(ctx, "failed to delete newly saved store logo after database create failure",
					"store_id", storeModel.ID,
					"logo_url", logoURL,
					"delete_error", deleteErr.Error(),
					"create_error", err.Error(),
				)
			}
		}
		return Store{}, err
	}
	return created, nil
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID string) (Store, error) {

	return s.repo.GetByID(ctx, storeID)
}

func (s Service) ListMyStores(ctx context.Context, actor auth.Claims) ([]Store, error) {
	return s.repo.ListByUser(ctx, actor.UserID, actor.Role)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID string, input UpdateStoreRequest) (Store, error) {

	current, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return Store{}, err
	}
	beforeSnapshot := activitySnapshot(current)

	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
	}
	if strings.TrimSpace(current.Name) == "" {
		return Store{}, ErrInvalidStoreName
	}

	if input.Phone != nil {
		current.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Fax != nil {
		current.Fax = strings.TrimSpace(*input.Fax)
	}
	if input.Email != nil {
		current.Email = strings.TrimSpace(*input.Email)
	}
	if input.Website != nil {
		current.Website = strings.TrimSpace(*input.Website)
	}
	if input.Address != nil {
		current.Address = strings.TrimSpace(*input.Address)
	}
	if input.PromptPayID != nil {
		current.PromptPayID = strings.TrimSpace(*input.PromptPayID)
	}
	if input.TaxID != nil {
		current.TaxID = strings.TrimSpace(*input.TaxID)
	}

	if input.CurrencyCode != nil {
		current.CurrencyCode = strings.ToUpper(strings.TrimSpace(*input.CurrencyCode))
	}
	if strings.TrimSpace(current.CurrencyCode) == "" {
		return Store{}, ErrInvalidCurrencyCode
	}

	oldLogoURL := current.LogoURL
	newLogoURL := ""
	if input.LogoFile != nil {
		logoURL, saveErr := s.storage.SaveStoreLogo(input.LogoFile)
		if saveErr != nil {
			return Store{}, saveErr
		}
		newLogoURL = logoURL
		current.LogoURL = logoURL
	}

	if err := s.repo.Update(ctx, storeID, current); err != nil {
		if newLogoURL != "" {
			if deleteErr := s.storage.DeleteStoreLogo(newLogoURL); deleteErr != nil {
				slog.WarnContext(ctx, "failed to delete newly saved store logo after database update failure",
					"store_id", storeID,
					"new_logo_url", newLogoURL,
					"delete_error", deleteErr.Error(),
					"update_error", err.Error(),
				)
			}
		}
		return Store{}, err
	}

	updated, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return Store{}, err
	}
	if newLogoURL != "" && oldLogoURL != "" && oldLogoURL != newLogoURL {
		if deleteErr := s.storage.DeleteStoreLogo(oldLogoURL); deleteErr != nil {
			slog.WarnContext(ctx, "failed to delete replaced store logo",
				"store_id", storeID,
				"old_logo_url", oldLogoURL,
				"new_logo_url", newLogoURL,
				"delete_error", deleteErr.Error(),
			)
		}
	}
	activitycapture.Record(ctx, "store", beforeSnapshot, activitySnapshot(updated))
	return updated, nil
}

func (s Service) ListBankAccounts(ctx context.Context, actor auth.Claims, storeID string) ([]StoreBankAccount, error) {
	// Cashiers need bank account details at checkout; allow any store member to list.
	return s.repo.ListBankAccounts(ctx, storeID)
}

func (s Service) CreateBankAccount(ctx context.Context, actor auth.Claims, storeID string, req CreateBankAccountRequest) (StoreBankAccount, error) {
	acc := StoreBankAccount{
		ID:          idgen.Generate(idgen.PrefixStoreBankAccount),
		StoreID:     storeID,
		BankCode:    strings.TrimSpace(req.BankCode),
		BankName:    strings.TrimSpace(req.BankName),
		AccountNo:   strings.TrimSpace(req.AccountNo),
		AccountName: strings.TrimSpace(req.AccountName),
		IsActive:    true,
	}
	return s.repo.CreateBankAccount(ctx, acc)
}

func (s Service) UpdateBankAccount(ctx context.Context, actor auth.Claims, storeID, id string, req UpdateBankAccountRequest) (StoreBankAccount, error) {
	updates := map[string]interface{}{}
	if req.BankName != nil {
		updates["bank_name"] = strings.TrimSpace(*req.BankName)
	}
	if req.AccountNo != nil {
		updates["account_no"] = strings.TrimSpace(*req.AccountNo)
	}
	if req.AccountName != nil {
		updates["account_name"] = strings.TrimSpace(*req.AccountName)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	return s.repo.UpdateBankAccount(ctx, storeID, id, updates)
}

func (s Service) DeleteBankAccount(ctx context.Context, actor auth.Claims, storeID, id string) error {
	return s.repo.DeleteBankAccount(ctx, storeID, id)
}
