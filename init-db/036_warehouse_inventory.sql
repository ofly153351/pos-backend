-- ============================================================
-- Warehouse Inventory System
-- Creates: warehouse_inventory (transferred stock awaiting allocation)
-- Alters: warehouses (add source_store_id, source_warehouse_id)
-- ============================================================

-- 1. Warehouse inventory — stock transferred from another store
--    but NOT yet allocated to saleable stock (stocks table).
CREATE TABLE IF NOT EXISTS warehouse_inventory (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    warehouse_id TEXT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    source_store_id TEXT REFERENCES stores(id) ON DELETE SET NULL,
    source_warehouse_id TEXT,
    transferred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT warehouse_inventory_wh_product_unique UNIQUE (warehouse_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_store ON warehouse_inventory (store_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_warehouse ON warehouse_inventory (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_product ON warehouse_inventory (product_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_source_store ON warehouse_inventory (source_store_id);

-- 2. Add source tracking columns to warehouses
ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS source_store_id TEXT REFERENCES stores(id) ON DELETE SET NULL;
ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS source_warehouse_id TEXT;

CREATE INDEX IF NOT EXISTS idx_warehouses_source_store ON warehouses (source_store_id);

-- 3. Add allocation type to stock_movements if not already present
--    (allocation moves from warehouse_inventory → stocks)
ALTER TABLE stock_movements
ADD COLUMN IF NOT EXISTS inventory_id TEXT REFERENCES warehouse_inventory(id) ON DELETE SET NULL;
