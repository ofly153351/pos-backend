ALTER TABLE products
ADD COLUMN IF NOT EXISTS barcode TEXT;

DROP VIEW IF EXISTS product_view;

CREATE VIEW product_view AS
SELECT
    p.id,
    p.store_id,
    p.product_type_id,
    COALESCE(pt.name, '') AS product_type_name,
    p.product_unit_id,
    COALESCE(pu.name, '') AS product_unit_name,
    p.brand_id,
    COALESCE(pb.name, '') AS brand_name,
    p.name,
    p.sku,
    p.barcode,
    p.image_url,
    p.min_stock,
    p.max_stock,
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
LEFT JOIN product_units pu ON pu.id = p.product_unit_id
LEFT JOIN product_brands pb ON pb.id = p.brand_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_store_barcode
ON products (store_id, barcode) WHERE barcode IS NOT NULL AND barcode != '';
