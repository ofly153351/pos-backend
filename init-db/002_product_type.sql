CREATE TABLE IF NOT EXISTS product_types (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_types_store_slug_unique UNIQUE (store_id, slug)
);

ALTER TABLE products
ADD COLUMN IF NOT EXISTS product_type_id TEXT REFERENCES product_types(id) ON DELETE SET NULL;

ALTER TABLE products
ADD COLUMN IF NOT EXISTS unit_type TEXT NOT NULL DEFAULT 'piece';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'products' AND column_name = 'product_type'
    ) THEN
        EXECUTE 'UPDATE products SET unit_type = product_type WHERE product_type IS NOT NULL';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_product_types_store_id ON product_types (store_id);
