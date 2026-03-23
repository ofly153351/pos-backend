package auth

import (
	"context"
	"testing"
)

type fakeUserRepository struct {
	created map[string]User
	stores  map[string]string
}

func (r *fakeUserRepository) Create(_ context.Context, user User) (User, error) {
	if _, exists := r.created[user.Email]; exists {
		return User{}, ErrEmailExists
	}
	r.created[user.Email] = user
	return user, nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (User, error) {
	user, ok := r.created[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) FindPrimaryStoreIDByUserID(_ context.Context, userID string) (string, error) {
	return r.stores[userID], nil
}

func TestRegisterAndLogin(t *testing.T) {
	repo := &fakeUserRepository{
		created: make(map[string]User),
		stores:  make(map[string]string),
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
}
