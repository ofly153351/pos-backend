CREATE TABLE IF NOT EXISTS warehouse_products (
    id TEXT PRIMARY KEY,
    warehouse_id TEXT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (warehouse_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_warehouse_products_warehouse_id ON warehouse_products (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_products_product_id ON warehouse_products (product_id);
