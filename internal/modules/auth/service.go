package auth

import (
	"context"
	"strings"
	"time"
)

type Service struct {
	repo   UserRepository
	tokens TokenManager
}

func NewService(repo UserRepository, tokens TokenManager) Service {
	return Service{
		repo:   repo,
		tokens: tokens,
	}
}

func (s Service) Register(ctx context.Context, input RegisterRequest) (AuthResponse, error) {
	// Public self-registration always receives the lowest global role. Elevated
	// roles (platform_admin/owner/manager) are NEVER assignable from the request
	// body — store ownership is granted via store membership when the user creates
	// a store (store.CreateWithOwner inserts an 'owner' member), not via users.role.
	input.Role = RoleCashier

	if err := validateRegister(input); err != nil {
		return AuthResponse{}, err
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return AuthResponse{}, err
	}

	user := User{
		ID:           newID(),
		Name:         strings.TrimSpace(input.Name),
		Email:        normalizeEmail(input.Email),
		Role:         input.Role,
		Status:       "active",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	user, err = s.repo.Create(ctx, user)
	if err != nil {
		return AuthResponse{}, err
	}

	token, err := s.tokens.Issue(user)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		User:        sanitizeUser(user),
		AccessToken: token,
		TokenType:   "Bearer",
	}, nil
}

func (s Service) Login(ctx context.Context, input LoginRequest) (AuthResponse, error) {
	if err := validateLogin(input); err != nil {
		return AuthResponse{}, err
	}

	user, err := s.repo.FindByEmail(ctx, input.Email)
	if err != nil || !verifyPassword(input.Password, user.PasswordHash) {
		return AuthResponse{}, ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(user)
	if err != nil {
		return AuthResponse{}, err
	}

	storeID, err := s.repo.FindPrimaryStoreIDByUserID(ctx, user.ID)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		User:        sanitizeUser(user),
		StoreID:     storeID,
		AccessToken: token,
		TokenType:   "Bearer",
	}, nil
}

func (s Service) Logout(ctx context.Context, actor Claims) error {
	if actor.UserID == "" {
		return ErrInvalidCredentials
	}
	return s.repo.IncrementTokenVersion(ctx, actor.UserID)
}

func sanitizeUser(user User) User {
	user.PasswordHash = ""
	return user
}
