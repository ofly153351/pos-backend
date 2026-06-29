package activity_log

import (
	"context"
	"encoding/json"
	"errors"

	"pos-backend/internal/modules/auth"
)

// ErrForbidden is returned when the actor is not a member of the requested store.
var ErrForbidden = errors.New("forbidden")

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

type LogRequest struct {
	StoreID    string
	UserID     string
	UserName   string
	Action     string
	Module     string
	ResourceID string
	Method     string
	Path       string
	IPAddress  string
	Changes    JSONText
}

func (s Service) Log(ctx context.Context, req LogRequest) {
	_ = s.repo.Create(ctx, ActivityLog{
		StoreID:    req.StoreID,
		UserID:     req.UserID,
		UserName:   req.UserName,
		Action:     req.Action,
		Module:     req.Module,
		ResourceID: req.ResourceID,
		Method:     req.Method,
		Path:       req.Path,
		IPAddress:  req.IPAddress,
		Changes:    req.Changes,
	})
}

func (s Service) List(ctx context.Context, actor auth.Claims, q ListQuery) (ListResponse, error) {
	ok, err := s.repo.UserCanOperateStore(ctx, q.StoreID, actor.UserID, actor.Role)
	if err != nil {
		return ListResponse{}, err
	}
	if !ok {
		return ListResponse{}, ErrForbidden
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 200 {
		q.Limit = 50
	}

	logs, total, err := s.repo.List(ctx, q)
	if err != nil {
		return ListResponse{}, err
	}

	items := make([]LogEntry, len(logs))
	for i, l := range logs {
		items[i] = LogEntry{
			ID:         l.ID,
			StoreID:    l.StoreID,
			UserID:     l.UserID,
			UserName:   l.UserName,
			Action:     l.Action,
			Module:     l.Module,
			ResourceID: l.ResourceID,
			Method:     l.Method,
			Path:       l.Path,
			IPAddress:  l.IPAddress,
			CreatedAt:  l.CreatedAt,
			// Derived (never stored) — kept identical to the filter logic.
			Severity: DeriveSeverity(l.Action, l.Module),
			Category: DeriveCategory(l.Action, l.Module),
			Changes:  json.RawMessage(l.Changes),
		}
	}
	return ListResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit}, nil
}
