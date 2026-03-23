ALTER TABLE customers
ADD COLUMN IF NOT EXISTS customer_level INTEGER NOT NULL DEFAULT 1;

ALTER TABLE customers
DROP CONSTRAINT IF EXISTS customers_customer_level_check;

ALTER TABLE customers
ADD CONSTRAINT customers_customer_level_check CHECK (customer_level > 0);

DROP INDEX IF EXISTS idx_customers_parent_id;

ALTER TABLE customers
DROP COLUMN IF EXISTS parent_customer_id;
