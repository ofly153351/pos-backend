package producttype

import (
	"context"
	"database/sql"
	"errors"
)

type Repository interface {
	Create(ctx context.Context, item ProductType) (ProductType, error)
	ListByStore(ctx context.Context, storeID string) ([]ProductType, error)
	GetByID(ctx context.Context, storeID, id string) (ProductType, error)
	Update(ctx context.Context, item ProductType) (ProductType, error)
	Delete(ctx context.Context, storeID, id string) error
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct{ db *sql.DB }

func NewPostgresRepository(db *sql.DB) PostgresRepository { return PostgresRepository{db: db} }

func (r PostgresRepository) Create(ctx context.Context, item ProductType) (ProductType, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO product_types (id, store_id, name, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $6)
	`, item.ID, item.StoreID, item.Name, item.Description, item.IsActive, item.CreatedAt)
	if err != nil {
		return ProductType{}, err
	}
	item.UpdatedAt = item.CreatedAt
	return item, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]ProductType, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, store_id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM product_types
		WHERE store_id = $1
		ORDER BY name ASC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ProductType
	for rows.Next() {
		var item ProductType
		if err := rows.Scan(&item.ID, &item.StoreID, &item.Name, &item.Description, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, id string) (ProductType, error) {
	var item ProductType
	err := r.db.QueryRowContext(ctx, `
		SELECT id, store_id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM product_types
		WHERE store_id = $1 AND id = $2
	`, storeID, id).Scan(&item.ID, &item.StoreID, &item.Name, &item.Description, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProductType{}, ErrProductTypeNotFound
		}
		return ProductType{}, err
	}
	return item, nil
}

func (r PostgresRepository) Update(ctx context.Context, item ProductType) (ProductType, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE product_types
		SET name = $3, description = NULLIF($4, ''), is_active = $5, updated_at = $6
		WHERE store_id = $1 AND id = $2
	`, item.StoreID, item.ID, item.Name, item.Description, item.IsActive, item.UpdatedAt)
	if err != nil {
		return ProductType{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ProductType{}, err
	}
	if affected == 0 {
		return ProductType{}, ErrProductTypeNotFound
	}
	return item, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM product_types WHERE store_id = $1 AND id = $2`, storeID, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductTypeNotFound
	}
	return nil
}

func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM store_members WHERE store_id = $1 AND user_id = $2 AND role IN ('owner', 'manager')`, storeID, userID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
