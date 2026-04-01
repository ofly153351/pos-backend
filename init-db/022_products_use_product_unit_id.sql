ALTER TABLE products
ADD COLUMN IF NOT EXISTS product_unit_id TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'products_product_unit_id_fkey'
    ) THEN
        ALTER TABLE products
            ADD CONSTRAINT products_product_unit_id_fkey
            FOREIGN KEY (product_unit_id) REFERENCES product_units(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
            AND table_name = 'products'
            AND column_name = 'unit_type'
    ) THEN
        INSERT INTO product_units (id, store_id, name, description, is_active, created_at, updated_at)
        SELECT DISTINCT
            'pu_' || SUBSTR(MD5(p.store_id || ':' || LOWER(TRIM(p.unit_type))), 1, 24),
            p.store_id,
            TRIM(p.unit_type),
            NULL,
            TRUE,
            NOW(),
            NOW()
        FROM products p
        WHERE COALESCE(TRIM(p.unit_type), '') <> ''
        ON CONFLICT (id) DO NOTHING;

        UPDATE products p
        SET product_unit_id = pu.id
        FROM product_units pu
        WHERE p.product_unit_id IS NULL
            AND pu.store_id = p.store_id
            AND LOWER(TRIM(pu.name)) = LOWER(TRIM(p.unit_type))
            AND COALESCE(TRIM(p.unit_type), '') <> '';
    END IF;
END $$;

INSERT INTO product_units (id, store_id, name, description, is_active, created_at, updated_at)
SELECT DISTINCT
    'pu_' || SUBSTR(MD5(p.store_id || ':unit'), 1, 24),
    p.store_id,
    'unit',
    NULL,
    TRUE,
    NOW(),
    NOW()
FROM products p
WHERE p.product_unit_id IS NULL
ON CONFLICT (id) DO NOTHING;

UPDATE products p
SET product_unit_id = 'pu_' || SUBSTR(MD5(p.store_id || ':unit'), 1, 24)
WHERE p.product_unit_id IS NULL;

ALTER TABLE products
ALTER COLUMN product_unit_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_products_product_unit_id ON products (product_unit_id);

ALTER TABLE products
DROP COLUMN IF EXISTS unit_type;
