package subscription

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

func (s Service) ListPlans(ctx context.Context) ([]Plan, error) {
	return s.repo.ListPlans(ctx)
}

func (s Service) AdminListAll(ctx context.Context) ([]StoreSubscription, error) {
	return s.repo.ListAll(ctx)
}

func (s Service) GetCurrentByStore(ctx context.Context, actor auth.Claims, storeID string) (StoreSubscription, error) {

	return s.repo.GetCurrentByStore(ctx, storeID)
}

func (s Service) ChangePlan(ctx context.Context, actor auth.Claims, storeID string, input ChangeSubscriptionRequest) (StoreSubscription, error) {
	if strings.TrimSpace(input.PlanCode) == "" {
		return StoreSubscription{}, ErrInvalidPlanCode
	}

	return s.repo.ChangePlan(ctx, storeID, strings.TrimSpace(input.PlanCode), time.Now().UTC())
}

func (s Service) AdminChangePlan(ctx context.Context, storeID string, input ChangeSubscriptionRequest) (StoreSubscription, error) {
	if strings.TrimSpace(input.PlanCode) == "" {
		return StoreSubscription{}, ErrInvalidPlanCode
	}

	return s.repo.ChangePlan(ctx, storeID, strings.TrimSpace(input.PlanCode), time.Now().UTC())
}

func (s Service) AdminUpdateStatus(ctx context.Context, storeID string, input UpdateSubscriptionStatusRequest) (StoreSubscription, error) {
	status := strings.TrimSpace(input.Status)
	if !isValidStatus(status) {
		return StoreSubscription{}, ErrInvalidStatus
	}

	return s.repo.UpdateStatus(ctx, storeID, status)
}

func isValidStatus(status string) bool {
	switch status {
	case "trialing", "active", "past_due", "cancelled", "expired":
		return true
	default:
		return false
	}
}
