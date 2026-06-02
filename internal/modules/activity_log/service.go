package activity_log

import "context"

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
	})
}

func (s Service) List(ctx context.Context, q ListQuery) (ListResponse, error) {
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
		}
	}
	return ListResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit}, nil
}
