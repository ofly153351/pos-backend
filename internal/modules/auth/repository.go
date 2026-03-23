package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user User) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindPrimaryStoreIDByUserID(ctx context.Context, userID string) (string, error)
}

type PostgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) PostgresUserRepository {
	return PostgresUserRepository{db: db}
}

func (r PostgresUserRepository) Create(ctx context.Context, user User) (User, error) {
	user.Email = normalizeEmail(user.Email)
	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailExists
		}
		return User{}, err
	}
	return user, nil
}

func (r PostgresUserRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Model(&User{}).
		Where("email = ?", normalizeEmail(email)).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

func (r PostgresUserRepository) FindPrimaryStoreIDByUserID(ctx context.Context, userID string) (string, error) {
	type storeMember struct {
		StoreID string `gorm:"column:store_id"`
	}

	var member storeMember
	err := r.db.WithContext(ctx).
		Table("store_members").
		Select("store_id").
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Limit(1).
		Take(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}

	return member.StoreID, nil
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
