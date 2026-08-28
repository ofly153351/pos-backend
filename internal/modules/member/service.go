package member

import (
	"context"
	"net/mail"
	"strings"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/platform/activitycapture"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) ListMembers(ctx context.Context, actor auth.Claims, storeID string) ([]Member, error) {
	if strings.TrimSpace(storeID) == "" {
		return nil, ErrStoreIDRequired
	}
	return s.repo.ListMembers(ctx, storeID)
}

func (s Service) AddMember(ctx context.Context, actor auth.Claims, storeID string, input AddMemberRequest) (Member, error) {
	if strings.TrimSpace(storeID) == "" {
		return Member{}, ErrStoreIDRequired
	}

	role := strings.TrimSpace(input.Role)
	if !isValidRole(role) {
		return Member{}, ErrInvalidRole
	}
	// Only an owner (or platform_admin) may grant the owner role.
	if role == RoleOwner && !auth.StoreAccessFromContext(ctx).IsOwner() {
		return Member{}, ErrOwnerOnly
	}

	email := normalizeEmail(input.Email)
	if email == "" {
		return Member{}, ErrEmailRequired
	}
	if _, perr := mail.ParseAddress(email); perr != nil {
		return Member{}, ErrEmailRequired
	}

	// Reuse an existing user (cross-store), else create a brand-new one.
	userID, err := s.repo.FindUserIDByEmail(ctx, email)
	if err != nil {
		return Member{}, err
	}
	if userID != "" {
		return s.repo.InsertMember(ctx, storeID, userID, role)
	}

	if strings.TrimSpace(input.Name) == "" {
		return Member{}, ErrNameRequired
	}
	if len(input.Password) < 8 {
		return Member{}, ErrPasswordTooShort
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return Member{}, err
	}
	return s.repo.CreateUserWithMember(ctx, storeID, input.Name, email, hash, role)
}

func (s Service) UpdateMember(ctx context.Context, actor auth.Claims, storeID, userID string, input UpdateMemberRequest) (Member, error) {
	if strings.TrimSpace(storeID) == "" {
		return Member{}, ErrStoreIDRequired
	}
	if input.Role == nil && input.Status == nil {
		return Member{}, ErrNothingToUpdate
	}

	target, err := s.repo.GetMember(ctx, storeID, userID)
	if err != nil {
		return Member{}, err
	}

	newRole := target.Role
	if input.Role != nil {
		r := strings.TrimSpace(*input.Role)
		if !isValidRole(r) {
			return Member{}, ErrInvalidRole
		}
		newRole = r
	}
	newStatus := target.Status
	if input.Status != nil {
		st := strings.TrimSpace(*input.Status)
		if !isValidStatus(st) {
			return Member{}, ErrInvalidStatus
		}
		newStatus = st
	}

	// Owner-role operations are owner-only: granting owner, or modifying an
	// existing owner (demote/suspend), requires the caller to be an owner.
	if (newRole == RoleOwner || target.Role == RoleOwner) && !auth.StoreAccessFromContext(ctx).IsOwner() {
		return Member{}, ErrOwnerOnly
	}
	// Don't let a caller suspend their own membership and lock themselves out.
	if userID == actor.UserID && newStatus == StatusSuspended {
		return Member{}, ErrCannotSelfSuspend
	}

	drop, err := s.wouldDropLastOwner(ctx, storeID, target, newRole, newStatus)
	if err != nil {
		return Member{}, err
	}
	if drop {
		return Member{}, ErrLastOwner
	}

	var rolePtr, statusPtr *string
	if input.Role != nil {
		rolePtr = &newRole
	}
	if input.Status != nil {
		statusPtr = &newStatus
	}
	beforeSnapshot := map[string]any{"role": target.Role, "status": target.Status}
	updated, err := s.repo.UpdateMember(ctx, storeID, userID, rolePtr, statusPtr)
	if err != nil {
		return Member{}, err
	}
	activitycapture.Record(ctx, "member", beforeSnapshot, map[string]any{"role": updated.Role, "status": updated.Status})
	return updated, nil
}

func (s Service) RemoveMember(ctx context.Context, actor auth.Claims, storeID, userID string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrStoreIDRequired
	}
	if userID == actor.UserID {
		return ErrCannotSelfRemove
	}

	target, err := s.repo.GetMember(ctx, storeID, userID)
	if err != nil {
		return err
	}
	// Removing an owner is owner-only.
	if target.Role == RoleOwner && !auth.StoreAccessFromContext(ctx).IsOwner() {
		return ErrOwnerOnly
	}

	drop, err := s.wouldDropLastOwner(ctx, storeID, target, "", "")
	if err != nil {
		return err
	}
	if drop {
		return ErrLastOwner
	}

	return s.repo.DeleteMember(ctx, storeID, userID)
}

// wouldDropLastOwner reports whether applying (newRole,newStatus) to target — or
// removing it, signalled by empty newRole/newStatus — would leave the store with
// zero active owners.
func (s Service) wouldDropLastOwner(ctx context.Context, storeID string, target Member, newRole, newStatus string) (bool, error) {
	isActiveOwner := target.Role == RoleOwner && target.Status == StatusActive
	willBeActiveOwner := newRole == RoleOwner && newStatus == StatusActive
	if !isActiveOwner || willBeActiveOwner {
		return false, nil
	}
	count, err := s.repo.CountActiveOwners(ctx, storeID)
	if err != nil {
		return false, err
	}
	return count <= 1, nil
}
