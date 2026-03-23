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
	role := strings.TrimSpace(input.Role)
	if role == "" {
		role = RoleOwner
	}
	input.Role = role

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

func sanitizeUser(user User) User {
	user.PasswordHash = ""
	return user
}
