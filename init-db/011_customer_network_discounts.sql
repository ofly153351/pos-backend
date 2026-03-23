ALTER TABLE sales
ADD COLUMN IF NOT EXISTS customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL;

ALTER TABLE sales
ADD COLUMN IF NOT EXISTS customer_level INTEGER;

ALTER TABLE sales
ADD COLUMN IF NOT EXISTS network_discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS customer_level_discounts (
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    level INTEGER NOT NULL CHECK (level > 0),
    discount_percent NUMERIC(5,2) NOT NULL CHECK (discount_percent >= 0 AND discount_percent <= 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (store_id, level)
);
