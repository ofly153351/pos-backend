-- 034_products_default_location.sql
-- Adds an AUTHORITATIVE per-product default receiving/storage location.
--
-- Background: products.storage_location is a free-text label (a display hint),
-- not a foreign key. Goods Receiving needs ONE authoritative location per product
-- that resolves to a real locations row, so stock and the IN movement land at a
-- deterministic location instead of an arbitrary "first" one.
--
-- This migration is idempotent (ADD COLUMN IF NOT EXISTS, CREATE OR REPLACE VIEW).
-- The column is NULLABLE: existing products may temporarily have no default
-- location after this migration (no guessed backfill). ON DELETE SET NULL keeps
-- products valid if the referenced location is later removed.

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS default_location_id TEXT REFERENCES locations(id) ON DELETE SET NULL;

-- Surface the new authoritative field on the product read model so it round-trips
-- through the product list/detail API. Columns are unchanged except the appended
-- default_location_id at the end (required by CREATE OR REPLACE VIEW).
CREATE OR REPLACE VIEW product_view AS
SELECT
    p.id,
    p.store_id,
    p.product_type_id,
    COALESCE(pt.name, '')  AS product_type_name,
    p.product_unit_id,
    COALESCE(pu.name, '')  AS product_unit_name,
    p.brand_id,
    COALESCE(pb.name, '')  AS brand_name,
    p.name,
    p.sku,
    p.barcode,
    p.product_code,
    p.description,
    p.storage_location,
    p.image_url,
    p.min_stock,
    p.max_stock,
    COALESCE((
        SELECT SUM(s.quantity)
        FROM stocks s
        JOIN locations l ON l.id = s.location_id AND l.is_sale_point = TRUE
        WHERE s.product_id = p.id
    ), 0)::integer AS total_stock,
    COALESCE((
        SELECT SUM(s.quantity)
        FROM stocks s
        WHERE s.product_id = p.id
    ), 0)::integer AS warehouse_stock,
    p.base_price,
    p.cost_price,
    p.special_price,
    p.special_price_start_at,
    p.special_price_end_at,
    p.is_active,
    p.created_at,
    p.updated_at,
    p.default_location_id
FROM products p
LEFT JOIN product_types  pt ON pt.id = p.product_type_id
LEFT JOIN product_units  pu ON pu.id = p.product_unit_id
LEFT JOIN product_brands pb ON pb.id = p.brand_id;
