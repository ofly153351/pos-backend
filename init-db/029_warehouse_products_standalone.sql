-- Make warehouse_products support standalone products (no product_id ref)
-- product_id becomes nullable: null = standalone warehouse product

-- Make product_id nullable (FK constraint remains for non-null values)
ALTER TABLE warehouse_products ALTER COLUMN product_id DROP NOT NULL;

-- Drop the UNIQUE constraint (product_id can be null now, can't have null in unique)
ALTER TABLE warehouse_products DROP CONSTRAINT IF EXISTS warehouse_products_warehouse_id_product_id_key;

-- Add partial unique index for non-null product_id pairs
CREATE UNIQUE INDEX IF NOT EXISTS idx_wp_warehouse_product_unique
ON warehouse_products (warehouse_id, product_id)
WHERE product_id IS NOT NULL;

-- Add standalone columns for products not referencing the products table
ALTER TABLE warehouse_products ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE warehouse_products ADD COLUMN IF NOT EXISTS sku TEXT;
ALTER TABLE warehouse_products ADD COLUMN IF NOT EXISTS barcode TEXT;
ALTER TABLE warehouse_products ADD COLUMN IF NOT EXISTS price DECIMAL(12,2) NOT NULL DEFAULT 0;
ALTER TABLE warehouse_products ADD COLUMN IF NOT EXISTS unit_name TEXT;
ALTER TABLE warehouse_products ADD COLUMN IF NOT EXISTS type_name TEXT;
ALTER TABLE warehouse_products ADD COLUMN IF NOT EXISTS image_url TEXT;
