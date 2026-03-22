ALTER TABLE product_types
DROP CONSTRAINT IF EXISTS product_types_store_slug_unique;

ALTER TABLE product_types
DROP COLUMN IF EXISTS slug;
