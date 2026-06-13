package member

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"pos-backend/internal/idgen"
)

type Repository interface {
	ListMembers(ctx context.Context, storeID string) ([]Member, error)
	GetMember(ctx context.Context, storeID, userID string) (Member, error)
	// GetActiveRole returns the user's STORE role only when the membership is
	// active; it returns "" (no error) when the user is not a member or is
	// suspended. Used to gate the caller.
	GetActiveRole(ctx context.Context, storeID, userID string) (string, error)
	CountActiveOwners(ctx context.Context, storeID string) (int, error)
	FindUserIDByEmail(ctx context.Context, email string) (string, error)
	InsertMember(ctx context.Context, storeID, userID, role string) (Member, error)
	CreateUserWithMember(ctx context.Context, storeID, name, email, passwordHash, role string) (Member, error)
	UpdateMember(ctx context.Context, storeID, userID string, role, status *string) (Member, error)
	DeleteMember(ctx context.Context, storeID, userID string) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

const memberSelect = `
	sm.user_id   AS user_id,
	u.full_name  AS full_name,
	u.email      AS email,
	sm.role      AS role,
	sm.status    AS status,
	sm.created_at AS created_at
`

func (r PostgresRepository) ListMembers(ctx context.Context, storeID string) ([]Member, error) {
	var items []Member
	err := r.db.WithContext(ctx).
		Table("store_members sm").
		Select(memberSelect).
		Joins("JOIN users u ON u.id = sm.user_id").
		Where("sm.store_id = ?", storeID).
		Order("sm.created_at ASC").
		Scan(&items).Error
	return items, err
}

func (r PostgresRepository) GetMember(ctx context.Context, storeID, userID string) (Member, error) {
	var item Member
	err := r.db.WithContext(ctx).
		Table("store_members sm").
		Select(memberSelect).
		Joins("JOIN users u ON u.id = sm.user_id").
		Where("sm.store_id = ? AND sm.user_id = ?", storeID, userID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Member{}, ErrMemberNotFound
		}
		return Member{}, err
	}
	return item, nil
}

func (r PostgresRepository) GetActiveRole(ctx context.Context, storeID, userID string) (string, error) {
	var row struct {
		Role string `gorm:"column:role"`
	}
	err := r.db.WithContext(ctx).
		Table("store_members").
		Select("role").
		Where("store_id = ? AND user_id = ? AND status <> 'suspended'", storeID, userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return row.Role, nil
}

func (r PostgresRepository) CountActiveOwners(ctx context.Context, storeID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND role = 'owner' AND status = 'active'", storeID).
		Count(&count).Error
	return int(count), err
}

func (r PostgresRepository) FindUserIDByEmail(ctx context.Context, email string) (string, error) {
	var row struct {
		ID string `gorm:"column:id"`
	}
	err := r.db.WithContext(ctx).
		Table("users").
		Select("id").
		Where("email = ?", normalizeEmail(email)).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return row.ID, nil
}

// InsertMember adds a membership for an EXISTING user. A duplicate (store_id,
// user_id) maps to ErrAlreadyMember.
func (r PostgresRepository) InsertMember(ctx context.Context, storeID, userID, role string) (Member, error) {
	payload := map[string]any{
		"id":         idgen.Generate(idgen.PrefixStoreMember),
		"store_id":   storeID,
		"user_id":    userID,
		"role":       role,
		"status":     StatusActive,
		"created_at": time.Now().UTC(),
	}
	if err := r.db.WithContext(ctx).Table("store_members").Create(payload).Error; err != nil {
		if isUniqueViolation(err) {
			return Member{}, ErrAlreadyMember
		}
		return Member{}, err
	}
	return r.GetMember(ctx, storeID, userID)
}

// CreateUserWithMember creates a brand-new user (global role 'cashier') and a
// membership for the store in one transaction.
func (r PostgresRepository) CreateUserWithMember(ctx context.Context, storeID, name, email, passwordHash, role string) (Member, error) {
	userID := idgen.Generate(idgen.PrefixUser)
	now := time.Now().UTC()

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userPayload := map[string]any{
			"id":            userID,
			"full_name":     strings.TrimSpace(name),
			"email":         normalizeEmail(email),
			"password_hash": passwordHash,
			"token_version": 0,
			// Global role is always the lowest tier; store power lives on the
			// membership row below.
			"role":       "cashier",
			"status":     "active",
			"created_at": now,
			"updated_at": now,
		}
		if err := tx.Table("users").Create(userPayload).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrAlreadyMember
			}
			return err
		}

		memberPayload := map[string]any{
			"id":         idgen.Generate(idgen.PrefixStoreMember),
			"store_id":   storeID,
			"user_id":    userID,
			"role":       role,
			"status":     StatusActive,
			"created_at": now,
		}
		if err := tx.Table("store_members").Create(memberPayload).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrAlreadyMember
			}
			return err
		}
		return nil
	})
	if err != nil {
		return Member{}, err
	}
	return r.GetMember(ctx, storeID, userID)
}

func (r PostgresRepository) UpdateMember(ctx context.Context, storeID, userID string, role, status *string) (Member, error) {
	updates := map[string]any{}
	if role != nil {
		updates["role"] = *role
	}
	if status != nil {
		updates["status"] = *status
	}
	if len(updates) == 0 {
		return r.GetMember(ctx, storeID, userID)
	}

	result := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ?", storeID, userID).
		Updates(updates)
	if result.Error != nil {
		return Member{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Member{}, ErrMemberNotFound
	}
	return r.GetMember(ctx, storeID, userID)
}

// DeleteMember removes the membership only — never the user row (the user may
// belong to other stores).
func (r PostgresRepository) DeleteMember(ctx context.Context, storeID, userID string) error {
	result := r.db.WithContext(ctx).
		Exec("DELETE FROM store_members WHERE store_id = ? AND user_id = ?", storeID, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
