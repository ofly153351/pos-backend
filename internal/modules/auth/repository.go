package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user User) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindPrimaryStoreIDByUserID(ctx context.Context, userID string) (string, error)
}

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) PostgresUserRepository {
	return PostgresUserRepository{db: db}
}

func (r PostgresUserRepository) Create(ctx context.Context, user User) (User, error) {
	query := `
		INSERT INTO users (id, full_name, email, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, full_name, email, role, status, created_at
	`

	row := r.db.QueryRowContext(
		ctx,
		query,
		user.ID,
		user.Name,
		normalizeEmail(user.Email),
		user.PasswordHash,
		user.Role,
		user.Status,
		user.CreatedAt,
	)

	var created User
	if err := row.Scan(&created.ID, &created.Name, &created.Email, &created.Role, &created.Status, &created.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailExists
		}
		return User{}, err
	}

	return created, nil
}

func (r PostgresUserRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	query := `
		SELECT id, full_name, email, password_hash, role, status, created_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, normalizeEmail(email)).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

func (r PostgresUserRepository) FindPrimaryStoreIDByUserID(ctx context.Context, userID string) (string, error) {
	query := `
		SELECT store_id
		FROM store_members
		WHERE user_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`

	var storeID string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&storeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	return storeID, nil
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
