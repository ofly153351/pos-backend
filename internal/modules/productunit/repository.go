package productunit

import (
	"context"
	"database/sql"
	"errors"
)

type Repository interface {
	Create(ctx context.Context, unit ProductUnit) (ProductUnit, error)
	ListByStore(ctx context.Context, storeID string) ([]ProductUnit, error)
	GetByID(ctx context.Context, storeID, id string) (ProductUnit, error)
	Update(ctx context.Context, unit ProductUnit) (ProductUnit, error)
	Delete(ctx context.Context, storeID, id string) error
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository { return PostgresRepository{db: db} }

func (r PostgresRepository) Create(ctx context.Context, unit ProductUnit) (ProductUnit, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO product_units (id, store_id, name, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $6)
	`, unit.ID, unit.StoreID, unit.Name, unit.Description, unit.IsActive, unit.CreatedAt)
	if err != nil {
		return ProductUnit{}, err
	}
	unit.UpdatedAt = unit.CreatedAt
	return unit, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]ProductUnit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, store_id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM product_units
		WHERE store_id = $1
		ORDER BY name ASC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ProductUnit
	for rows.Next() {
		var unit ProductUnit
		if err := rows.Scan(&unit.ID, &unit.StoreID, &unit.Name, &unit.Description, &unit.IsActive, &unit.CreatedAt, &unit.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, unit)
	}
	return list, rows.Err()
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, id string) (ProductUnit, error) {
	var unit ProductUnit
	err := r.db.QueryRowContext(ctx, `
		SELECT id, store_id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM product_units
		WHERE store_id = $1 AND id = $2
	`, storeID, id).Scan(&unit.ID, &unit.StoreID, &unit.Name, &unit.Description, &unit.IsActive, &unit.CreatedAt, &unit.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProductUnit{}, ErrProductUnitNotFound
		}
		return ProductUnit{}, err
	}
	return unit, nil
}

func (r PostgresRepository) Update(ctx context.Context, unit ProductUnit) (ProductUnit, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE product_units
		SET name = $3, description = NULLIF($4, ''), is_active = $5, updated_at = $6
		WHERE store_id = $1 AND id = $2
	`, unit.StoreID, unit.ID, unit.Name, unit.Description, unit.IsActive, unit.UpdatedAt)
	if err != nil {
		return ProductUnit{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ProductUnit{}, err
	}
	if affected == 0 {
		return ProductUnit{}, ErrProductUnitNotFound
	}
	return unit, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM product_units WHERE store_id = $1 AND id = $2`, storeID, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductUnitNotFound
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
