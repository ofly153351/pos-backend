-- 064_product_default_warehouse_stock.sql
-- Ensure every product has an explicit zero-balance stock row in the store's
-- default warehouse. This assigns ownership only; it does not move quantity
-- or create a stock movement.

INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
SELECT
    'stk-default-' || md5(p.id || ':' || l.id),
    p.store_id,
    p.id,
    l.id,
    0,
    NOW(),
    NOW()
FROM products p
JOIN warehouses w
  ON w.store_id = p.store_id
 AND w.is_default = TRUE
 AND w.is_active = TRUE
JOIN LATERAL (
    SELECT l0.id
    FROM locations l0
    WHERE l0.store_id = p.store_id
      AND l0.warehouse_id = w.id
      AND l0.is_active = TRUE
    ORDER BY l0.is_sale_point ASC, l0.created_at ASC, l0.id ASC
    LIMIT 1
) l ON TRUE
WHERE NOT EXISTS (
    SELECT 1
    FROM stocks s
    JOIN locations existing_location ON existing_location.id = s.location_id
    WHERE s.product_id = p.id
      AND existing_location.warehouse_id = w.id
)
ON CONFLICT (product_id, location_id) DO NOTHING;
