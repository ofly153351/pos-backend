ALTER TABLE products
DROP COLUMN IF EXISTS product_type;

ALTER TABLE products
DROP COLUMN IF EXISTS unit_type;

CREATE OR REPLACE VIEW product_view AS
SELECT
    p.id,
    p.store_id,
    p.product_type_id,
    COALESCE(pt.name, '') AS product_type_name,
    p.product_unit_id,
    COALESCE(pu.name, '') AS product_unit_name,
    p.name,
    p.sku,
    p.image_url,
    p.quantity,
    p.base_price,
    p.special_price,
    p.special_price_start_at,
    p.special_price_end_at,
    p.is_active,
    p.created_at,
    p.updated_at
FROM products p
LEFT JOIN product_types pt ON pt.id = p.product_type_id
LEFT JOIN product_units pu ON pu.id = p.product_unit_id;
