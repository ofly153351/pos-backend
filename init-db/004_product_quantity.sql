ALTER TABLE products
ADD COLUMN IF NOT EXISTS quantity INTEGER NOT NULL DEFAULT 0;

ALTER TABLE products
DROP CONSTRAINT IF EXISTS products_quantity_check;

ALTER TABLE products
ADD CONSTRAINT products_quantity_check CHECK (quantity >= 0);
