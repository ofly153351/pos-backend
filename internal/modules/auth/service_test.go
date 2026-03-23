package auth

import (
	"context"
	"testing"
)

type fakeUserRepository struct {
	created       map[string]User
	stores        map[string]string
	tokenVersions map[string]int64
}

func (r *fakeUserRepository) Create(_ context.Context, user User) (User, error) {
	if _, exists := r.created[user.Email]; exists {
		return User{}, ErrEmailExists
	}
	user.TokenVersion = 0
	r.created[user.Email] = user
	r.tokenVersions[user.ID] = 0
	return user, nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (User, error) {
	user, ok := r.created[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	user.TokenVersion = r.tokenVersions[user.ID]
	return user, nil
}

func (r *fakeUserRepository) FindPrimaryStoreIDByUserID(_ context.Context, userID string) (string, error) {
	return r.stores[userID], nil
}

func (r *fakeUserRepository) FindTokenVersionByUserID(_ context.Context, userID string) (int64, error) {
	version, ok := r.tokenVersions[userID]
	if !ok {
		return 0, ErrUserNotFound
	}
	return version, nil
}

func (r *fakeUserRepository) IncrementTokenVersion(_ context.Context, userID string) error {
	_, ok := r.tokenVersions[userID]
	if !ok {
		return ErrUserNotFound
	}
	r.tokenVersions[userID]++
	return nil
}

func TestRegisterAndLogin(t *testing.T) {
	repo := &fakeUserRepository{
		created:       make(map[string]User),
		stores:        make(map[string]string),
		tokenVersions: make(map[string]int64),
	}
	service := NewService(repo, NewTokenManager("test-secret", 24))

	registerRes, err := service.Register(context.Background(), RegisterRequest{
		Name:     "POS Admin",
		Email:    "admin@example.com",
		Password: "password123",
		Role:     RoleOwner,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if registerRes.User.Role != RoleOwner {
		t.Fatalf("unexpected role: %s", registerRes.User.Role)
	}

	repo.stores[registerRes.User.ID] = "store_123"

	loginRes, err := service.Login(context.Background(), LoginRequest{
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if loginRes.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if loginRes.StoreID != "store_123" {
		t.Fatalf("unexpected store id: %s", loginRes.StoreID)
	}

	initialClaims, err := service.tokens.Parse(loginRes.AccessToken)
	if err != nil {
		t.Fatalf("parse initial token failed: %v", err)
	}
	if initialClaims.TokenVer != 0 {
		t.Fatalf("unexpected initial token version: %d", initialClaims.TokenVer)
	}

	if err := service.Logout(context.Background(), Claims{UserID: registerRes.User.ID}); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	loginResAfterLogout, err := service.Login(context.Background(), LoginRequest{
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login after logout failed: %v", err)
	}
	newClaims, err := service.tokens.Parse(loginResAfterLogout.AccessToken)
	if err != nil {
		t.Fatalf("parse token after logout failed: %v", err)
	}
	if newClaims.TokenVer != 1 {
		t.Fatalf("expected token version 1 after logout, got %d", newClaims.TokenVer)
	}
}
