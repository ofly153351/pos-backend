CREATE TABLE IF NOT EXISTS warehouses (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT,
    address TEXT,
    phone TEXT,
    contact_name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT warehouses_store_code_unique UNIQUE (store_id, code)
);

CREATE INDEX IF NOT EXISTS idx_warehouses_store_id ON warehouses (store_id);
