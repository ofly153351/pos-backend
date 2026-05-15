CREATE TABLE IF NOT EXISTS product_brands (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_brands_store_name_unique UNIQUE (store_id, name)
);

CREATE INDEX IF NOT EXISTS idx_product_brands_store_id ON product_brands (store_id);

ALTER TABLE products
ADD COLUMN IF NOT EXISTS brand_id TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'products_brand_id_fkey'
    ) THEN
        ALTER TABLE products
            ADD CONSTRAINT products_brand_id_fkey
            FOREIGN KEY (brand_id) REFERENCES product_brands(id) ON DELETE SET NULL;
    END IF;
END $$;

INSERT INTO product_brands (id, store_id, name, is_active, created_at, updated_at)
SELECT
    'br_' || SUBSTR(MD5(p.store_id || ':' || LOWER(TRIM(p.brand))), 1, 24) AS id,
    p.store_id,
    TRIM(p.brand) AS name,
    TRUE,
    NOW(),
    NOW()
FROM products p
WHERE p.brand IS NOT NULL
  AND TRIM(p.brand) <> ''
ON CONFLICT (store_id, name) DO NOTHING;

UPDATE products p
SET brand_id = b.id
FROM product_brands b
WHERE p.store_id = b.store_id
  AND p.brand IS NOT NULL
  AND TRIM(p.brand) <> ''
  AND LOWER(TRIM(p.brand)) = LOWER(b.name)
  AND p.brand_id IS NULL;

DROP VIEW IF EXISTS product_view;

CREATE VIEW product_view AS
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
    p.updated_at,
    p.brand_id,
    COALESCE(pb.name, '') AS brand_name
FROM products p
LEFT JOIN product_types pt ON pt.id = p.product_type_id
LEFT JOIN product_units pu ON pu.id = p.product_unit_id
LEFT JOIN product_brands pb ON pb.id = p.brand_id;
