ALTER TABLE product_units
DROP CONSTRAINT IF EXISTS product_units_store_code_unique;

ALTER TABLE product_units
DROP COLUMN IF EXISTS code;
