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
		Slug:         buildSlug(input.Name, input.Slug),
		LogoURL:      logoURL,
		Phone:        strings.TrimSpace(input.Phone),
		Address:      strings.TrimSpace(input.Address),
		CurrencyCode: strings.ToUpper(strings.TrimSpace(input.CurrencyCode)),
		CreatedAt:    time.Now().UTC(),
	}

	return s.repo.CreateWithOwner(ctx, storeModel, actor.UserID, strings.TrimSpace(input.SubscriptionPlanCode))
}

func (s Service) CanManageStore(ctx context.Context, actor auth.Claims, storeID string) (bool, error) {
	return s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
}
