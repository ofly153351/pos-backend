package promotion

import (
	"context"
	"encoding/json"
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

func (s Service) ensureAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrStoreIDRequired
	}
	ok, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// ensureManageAccess gates write operations (create/update/delete) to owner/manager
// (or platform admin). Cashiers can read promotions but cannot mint/alter them —
// this prevents a cashier from fabricating an 'active' promotion to feed an
// unearned promo_discount through checkout.
func (s Service) ensureManageAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrStoreIDRequired
	}
	ok, err := s.repo.UserCanManageStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s Service) List(ctx context.Context, actor auth.Claims, storeID string) ([]json.RawMessage, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, storeID)
}

// Create stores the full campaign JSON verbatim; the server owns the id and a
// default status. Returns the stored campaign (with the server id) as raw JSON.
func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, body []byte) (json.RawMessage, error) {
	if err := s.ensureManageAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	obj := map[string]any{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, ErrInvalidBody
	}
	id := newPromotionID()
	obj["id"] = id
	if st, _ := obj["status"].(string); strings.TrimSpace(st) == "" {
		obj["status"] = "draft"
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, ErrInvalidBody
	}
	meta := extractMeta(data)
	now := time.Now().UTC().Format(time.RFC3339)
	row := PromotionRow{
		ID:        id,
		StoreID:   storeID,
		Name:      meta.Name,
		Type:      meta.Type,
		Status:    meta.Status,
		Code:      meta.Code,
		Data:      string(data),
		CreatedBy: actor.UserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Insert(ctx, row); err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func (s Service) Update(ctx context.Context, actor auth.Claims, storeID, id string, body []byte) (json.RawMessage, error) {
	if err := s.ensureManageAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, ErrIDRequired
	}
	obj := map[string]any{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, ErrInvalidBody
	}
	obj["id"] = id
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, ErrInvalidBody
	}
	meta := extractMeta(data)
	affected, err := s.repo.Update(ctx, storeID, id, map[string]any{
		"name":       meta.Name,
		"type":       meta.Type,
		"status":     meta.Status,
		"code":       meta.Code,
		"data":       string(data),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrNotFound
	}
	return json.RawMessage(data), nil
}

func (s Service) Delete(ctx context.Context, actor auth.Claims, storeID, id string) error {
	if err := s.ensureManageAccess(ctx, actor, storeID); err != nil {
		return err
	}
	affected, err := s.repo.Delete(ctx, storeID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
