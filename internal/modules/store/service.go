package store

import (
	"context"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
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

	return s.repo.CreateWithOwner(ctx, storeModel, actor.UserID, strings.TrimSpace(input.SubscriptionPlanCode))
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID string) (Store, error) {
	ok, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Store{}, err
	}
	if !ok {
		return Store{}, ErrStoreForbidden
	}

	return s.repo.GetByID(ctx, storeID)
}

func (s Service) ListMyStores(ctx context.Context, actor auth.Claims) ([]Store, error) {
	return s.repo.ListByUser(ctx, actor.UserID, actor.Role)
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID string, input UpdateStoreRequest) (Store, error) {
	ok, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Store{}, err
	}
	if !ok {
		return Store{}, ErrStoreForbidden
	}

	current, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return Store{}, err
	}

	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
	}
	if strings.TrimSpace(current.Name) == "" {
		return Store{}, ErrInvalidStoreName
	}

	if input.Phone != nil {
		current.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Address != nil {
		current.Address = strings.TrimSpace(*input.Address)
	}
	if input.PromptPayID != nil {
		current.PromptPayID = strings.TrimSpace(*input.PromptPayID)
	}

	if input.CurrencyCode != nil {
		current.CurrencyCode = strings.ToUpper(strings.TrimSpace(*input.CurrencyCode))
	}
	if strings.TrimSpace(current.CurrencyCode) == "" {
		return Store{}, ErrInvalidCurrencyCode
	}

	if input.LogoFile != nil {
		logoURL, saveErr := s.storage.SaveStoreLogo(input.LogoFile)
		if saveErr != nil {
			return Store{}, saveErr
		}
		current.LogoURL = logoURL
	}

	if err := s.repo.Update(ctx, storeID, current); err != nil {
		return Store{}, err
	}

	return s.repo.GetByID(ctx, storeID)
}

func (s Service) CanManageStore(ctx context.Context, actor auth.Claims, storeID string) (bool, error) {
	return s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
}
