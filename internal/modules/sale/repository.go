package sale

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Repository interface {
	Create(ctx context.Context, sale Sale) (Sale, error)
	ListByStore(ctx context.Context, storeID string) ([]Sale, error)
	GetByID(ctx context.Context, storeID, saleID string) (Sale, error)
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Create(ctx context.Context, sale Sale) (Sale, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Sale{}, err
	}
	defer tx.Rollback()

	for index, item := range sale.Items {
		product, err := r.lockProductForSale(ctx, tx, sale.StoreID, item.ProductID)
		if err != nil {
			return Sale{}, err
		}
		if !product.IsActive {
			return Sale{}, ErrProductInactive
		}
		if product.Quantity < item.Quantity {
			return Sale{}, fmt.Errorf("%w for product %s", ErrInsufficientStock, item.ProductID)
		}

		unitPrice := resolveEffectivePrice(product, sale.SoldAt)
		discountAmountPerUnit, err := calculateDiscount(item.DiscountType, item.DiscountValue, unitPrice)
		if err != nil {
			return Sale{}, err
		}

		sale.Items[index].ID = newID()
		sale.Items[index].SaleID = sale.ID
		sale.Items[index].ProductName = product.Name
		sale.Items[index].SKU = product.SKU
		sale.Items[index].UnitType = product.UnitType
		sale.Items[index].UnitPrice = unitPrice
		sale.Items[index].DiscountType = normalizeDiscountType(item.DiscountType)
		sale.Items[index].DiscountAmountPerUnit = discountAmountPerUnit
		sale.Items[index].LineSubtotal = unitPrice * float64(item.Quantity)
		sale.Items[index].LineDiscountTotal = discountAmountPerUnit * float64(item.Quantity)
		sale.Items[index].LineTotal = unitPrice * float64(item.Quantity)
		sale.Items[index].LineTotal -= sale.Items[index].LineDiscountTotal
		sale.Items[index].CreatedAt = sale.CreatedAt
		sale.SubtotalAmount += sale.Items[index].LineSubtotal
		sale.DiscountAmount += sale.Items[index].LineDiscountTotal
		sale.TotalAmount += sale.Items[index].LineTotal

		if _, err := tx.ExecContext(
			ctx,
			`UPDATE products SET quantity = quantity - $3, updated_at = $4 WHERE store_id = $1 AND id = $2`,
			sale.StoreID,
			product.ID,
			item.Quantity,
			sale.CreatedAt,
		); err != nil {
			return Sale{}, err
		}
	}

	sale.ChangeAmount = sale.PaidAmount - sale.TotalAmount
	if sale.ChangeAmount < 0 {
		return Sale{}, ErrInvalidPaidAmount
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO sales (id, store_id, sale_number, cashier_user_id, status, payment_method, note, total_items, subtotal_amount, discount_amount, total_amount, paid_amount, change_amount, sold_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10, $11, $12, $13, $14, $15)`,
		sale.ID,
		sale.StoreID,
		sale.SaleNumber,
		sale.CashierUserID,
		sale.Status,
		sale.PaymentMethod,
		sale.Note,
		sale.TotalItems,
		sale.SubtotalAmount,
		sale.DiscountAmount,
		sale.TotalAmount,
		sale.PaidAmount,
		sale.ChangeAmount,
		sale.SoldAt,
		sale.CreatedAt,
	); err != nil {
		return Sale{}, err
	}

	for _, item := range sale.Items {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO sale_items (id, sale_id, product_id, product_name, sku, unit_type, quantity, unit_price, discount_type, discount_value, discount_amount_per_unit, line_subtotal, line_discount_total, line_total, created_at)
			 VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, NULLIF($9, ''), $10, $11, $12, $13, $14, $15)`,
			item.ID,
			item.SaleID,
			item.ProductID,
			item.ProductName,
			item.SKU,
			item.UnitType,
			item.Quantity,
			item.UnitPrice,
			item.DiscountType,
			item.DiscountValue,
			item.DiscountAmountPerUnit,
			item.LineSubtotal,
			item.LineDiscountTotal,
			item.LineTotal,
			item.CreatedAt,
		); err != nil {
			return Sale{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Sale{}, err
	}

	return sale, nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string) ([]Sale, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, store_id, sale_number, cashier_user_id, status, COALESCE(payment_method, ''), COALESCE(note, ''), total_items, subtotal_amount, discount_amount, total_amount, paid_amount, change_amount, sold_at, created_at
		FROM sales
		WHERE store_id = $1
		ORDER BY sold_at DESC, created_at DESC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sales []Sale
	for rows.Next() {
		var item Sale
		if err := rows.Scan(
			&item.ID,
			&item.StoreID,
			&item.SaleNumber,
			&item.CashierUserID,
			&item.Status,
			&item.PaymentMethod,
			&item.Note,
			&item.TotalItems,
			&item.SubtotalAmount,
			&item.DiscountAmount,
			&item.TotalAmount,
			&item.PaidAmount,
			&item.ChangeAmount,
			&item.SoldAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		sales = append(sales, item)
	}
	return sales, rows.Err()
}

func (r PostgresRepository) GetByID(ctx context.Context, storeID, saleID string) (Sale, error) {
	var sale Sale
	err := r.db.QueryRowContext(ctx, `
		SELECT id, store_id, sale_number, cashier_user_id, status, COALESCE(payment_method, ''), COALESCE(note, ''), total_items, subtotal_amount, discount_amount, total_amount, paid_amount, change_amount, sold_at, created_at
		FROM sales
		WHERE store_id = $1 AND id = $2
	`, storeID, saleID).Scan(
		&sale.ID,
		&sale.StoreID,
		&sale.SaleNumber,
		&sale.CashierUserID,
		&sale.Status,
		&sale.PaymentMethod,
		&sale.Note,
		&sale.TotalItems,
		&sale.SubtotalAmount,
		&sale.DiscountAmount,
		&sale.TotalAmount,
		&sale.PaidAmount,
		&sale.ChangeAmount,
		&sale.SoldAt,
		&sale.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Sale{}, ErrSaleNotFound
		}
		return Sale{}, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, sale_id, product_id, product_name, COALESCE(sku, ''), COALESCE(unit_type, ''), quantity, unit_price, COALESCE(discount_type, ''), discount_value, discount_amount_per_unit, line_subtotal, line_discount_total, line_total, created_at
		FROM sale_items
		WHERE sale_id = $1
		ORDER BY created_at ASC
	`, sale.ID)
	if err != nil {
		return Sale{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(
			&item.ID,
			&item.SaleID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.UnitType,
			&item.Quantity,
			&item.UnitPrice,
			&item.DiscountType,
			&item.DiscountValue,
			&item.DiscountAmountPerUnit,
			&item.LineSubtotal,
			&item.LineDiscountTotal,
			&item.LineTotal,
			&item.CreatedAt,
		); err != nil {
			return Sale{}, err
		}
		sale.Items = append(sale.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Sale{}, err
	}

	return sale, nil
}

func (r PostgresRepository) UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var exists int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM store_members WHERE store_id = $1 AND user_id = $2 AND role IN ('owner', 'manager', 'cashier')`,
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

func (r PostgresRepository) lockProductForSale(ctx context.Context, tx *sql.Tx, storeID, productID string) (productSnapshot, error) {
	var product productSnapshot
	err := tx.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(sku, ''), COALESCE(unit_type, ''), quantity, is_active, base_price, special_price, special_price_start_at, special_price_end_at
		FROM products
		WHERE store_id = $1 AND id = $2
		FOR UPDATE
	`, storeID, productID).Scan(
		&product.ID,
		&product.Name,
		&product.SKU,
		&product.UnitType,
		&product.Quantity,
		&product.IsActive,
		&product.BasePrice,
		&product.SpecialPrice,
		&product.SpecialPriceStartAt,
		&product.SpecialPriceEndAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return productSnapshot{}, ErrProductNotFound
		}
		return productSnapshot{}, err
	}
	return product, nil
}

func calculateDiscount(discountType string, discountValue *float64, unitPrice float64) (float64, error) {
	normalizedType := normalizeDiscountType(discountType)
	if normalizedType == "" {
		if discountValue != nil {
			return 0, ErrInvalidDiscountType
		}
		return 0, nil
	}
	if discountValue == nil {
		return 0, ErrDiscountValueRequired
	}
	if *discountValue < 0 {
		return 0, ErrInvalidDiscountValue
	}

	switch normalizedType {
	case DiscountTypeAmount:
		if *discountValue > unitPrice {
			return 0, ErrAmountDiscountExceedsPrice
		}
		return *discountValue, nil
	case DiscountTypePercent:
		if *discountValue > 100 {
			return 0, ErrInvalidPercentDiscount
		}
		return unitPrice * (*discountValue / 100), nil
	default:
		return 0, ErrInvalidDiscountType
	}
}
