package product

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Repository interface {
	Create(ctx context.Context, product Product) (Product, error)
	ListByStore(ctx context.Context, storeID string) ([]Product, error)
	GetByID(ctx context.Context, storeID, productID string) (Product, error)
	Update(ctx context.Context, product Product) (Product, error)
	Delete(ctx context.Context, storeID, productID string) error
	ProductTypeExists(ctx context.Context, storeID, productTypeID string) (bool, error)
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, product Product) (Product, error) {
	query := `
		INSERT INTO products (
			id, store_id, product_type_id, name, sku, unit_type, image_url, quantity, base_price, special_price, special_price_start_at, special_price_end_at, is_active, created_at, updated_at
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, NULLIF($7, ''), $8, $9, $10, $11, $12, $13, $14, $14)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		product.ID,
		product.StoreID,
		product.ProductTypeID,
		product.Name,
		product.SKU,
		product.UnitType,
		product.ImageURL,
		product.Quantity,
		product.BasePrice,
		product.SpecialPrice,
		product.SpecialPriceStartAt,
		product.SpecialPriceEndAt,
		product.IsActive,
		product.CreatedAt,
	)
	if err != nil {
		return Product{}, err
	}

	product.EffectivePrice = resolveEffectivePrice(product, time.Now().UTC())
	product.UpdatedAt = product.CreatedAt
	return product, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Product, error) {
	query := `
		SELECT p.id, p.store_id, COALESCE(p.product_type_id, ''), COALESCE(pt.name, ''), p.name, COALESCE(p.sku, ''), p.unit_type, COALESCE(p.image_url, ''), p.quantity, p.base_price, p.special_price, p.special_price_start_at, p.special_price_end_at, p.is_active, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN product_types pt ON pt.id = p.product_type_id
		WHERE p.store_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	now := time.Now().UTC()
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		product.EffectivePrice = resolveEffectivePrice(product, now)
		products = append(products, product)
	}

	return products, rows.Err()
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, productID string) (Product, error) {
	query := `
		SELECT p.id, p.store_id, COALESCE(p.product_type_id, ''), COALESCE(pt.name, ''), p.name, COALESCE(p.sku, ''), p.unit_type, COALESCE(p.image_url, ''), p.quantity, p.base_price, p.special_price, p.special_price_start_at, p.special_price_end_at, p.is_active, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN product_types pt ON pt.id = p.product_type_id
		WHERE p.store_id = $1 AND p.id = $2
	`

	row := r.db.QueryRowContext(ctx, query, storeID, productID)
	product, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrProductNotFound
		}
		return Product{}, err
	}

	product.EffectivePrice = resolveEffectivePrice(product, time.Now().UTC())
	return product, nil
}

func (r PostgresRepository) Update(ctx context.Context, product Product) (Product, error) {
	query := `
		UPDATE products
		SET product_type_id = NULLIF($3, ''),
		    name = $4,
		    sku = NULLIF($5, ''),
		    unit_type = $6,
		    image_url = NULLIF($7, ''),
		    quantity = $8,
		    base_price = $9,
		    special_price = $10,
		    special_price_start_at = $11,
		    special_price_end_at = $12,
		    is_active = $13,
		    updated_at = $14
		WHERE store_id = $1 AND id = $2
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		product.StoreID,
		product.ID,
		product.ProductTypeID,
		product.Name,
		product.SKU,
		product.UnitType,
		product.ImageURL,
		product.Quantity,
		product.BasePrice,
		product.SpecialPrice,
		product.SpecialPriceStartAt,
		product.SpecialPriceEndAt,
		product.IsActive,
		product.UpdatedAt,
	)
	if err != nil {
		return Product{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Product{}, err
	}
	if affected == 0 {
		return Product{}, ErrProductNotFound
	}

	product.EffectivePrice = resolveEffectivePrice(product, time.Now().UTC())
	return product, nil
}

func (r PostgresRepository) Delete(ctx context.Context, storeID, productID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE store_id = $1 AND id = $2`, storeID, productID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r PostgresRepository) ProductTypeExists(ctx context.Context, storeID, productTypeID string) (bool, error) {
	if productTypeID == "" {
		return true, nil
	}
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM product_types WHERE store_id = $1 AND id = $2`, storeID, productTypeID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var exists int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM store_members WHERE store_id = $1 AND user_id = $2 AND role IN ('owner', 'manager')`,
		storeID,
		userID,
	).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProduct(row scanner) (Product, error) {
	var product Product
	err := row.Scan(
		&product.ID,
		&product.StoreID,
		&product.ProductTypeID,
		&product.ProductTypeName,
		&product.Name,
		&product.SKU,
		&product.UnitType,
		&product.ImageURL,
		&product.Quantity,
		&product.BasePrice,
		&product.SpecialPrice,
		&product.SpecialPriceStartAt,
		&product.SpecialPriceEndAt,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return Product{}, err
	}
	return product, nil
}
