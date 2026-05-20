-- ============================================================
-- Stock System Refactor
-- Creates: locations, stocks
-- Drops: warehouse_products (replaced by stocks + locations)
-- Alters: stock_movements (add location_id, expand types)
-- Alters: products (drop quantity)
-- ============================================================

-- 1. Storage locations (belongs to warehouse)
CREATE TABLE IF NOT EXISTS locations (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    warehouse_id TEXT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT,
    is_sale_point BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT locations_store_warehouse_code_unique UNIQUE (store_id, warehouse_id, code)
);

CREATE INDEX IF NOT EXISTS idx_locations_store_id ON locations (store_id);
CREATE INDEX IF NOT EXISTS idx_locations_warehouse_id ON locations (warehouse_id);

-- 2. Stock quantities (product per location)
CREATE TABLE IF NOT EXISTS stocks (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    location_id TEXT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stocks_product_location_unique UNIQUE (product_id, location_id)
);

CREATE INDEX IF NOT EXISTS idx_stocks_store_id ON stocks (store_id);
CREATE INDEX IF NOT EXISTS idx_stocks_product_id ON stocks (product_id);
CREATE INDEX IF NOT EXISTS idx_stocks_location_id ON stocks (location_id);

-- 3. Add location_id to stock_movements
ALTER TABLE stock_movements
ADD COLUMN IF NOT EXISTS location_id TEXT REFERENCES locations(id) ON DELETE SET NULL;

ALTER TABLE stock_movements
ADD COLUMN IF NOT EXISTS destination_location_id TEXT REFERENCES locations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_stock_movements_location ON stock_movements(location_id);

-- 4. Remove outdated tables and columns
DROP TABLE IF EXISTS warehouse_products;

-- 5. Remove quantity from products (now tracked in stocks table)
-- First ensure no FK references to products.quantity are broken
ALTER TABLE products DROP COLUMN IF EXISTS quantity;
