-- ============================================================
-- Add total_stock to product_view (aggregate from stocks table)
-- Run AFTER 034_stock_system.sql
-- ============================================================

DROP VIEW IF EXISTS product_view;

CREATE VIEW product_view AS
SELECT
    p.id,
    p.store_id,
    p.product_type_id,
    COALESCE(pt.name, '') AS product_type_name,
    p.product_unit_id,
    COALESCE(pu.name, '') AS product_unit_name,
    p.brand_id,
    COALESCE(pb.name, '') AS brand_name,
    p.name,
    p.sku,
    p.barcode,
    p.image_url,
    p.min_stock,
    p.max_stock,
    COALESCE(SUM(s.quantity), 0)::integer AS total_stock,
    p.base_price,
    p.special_price,
    p.special_price_start_at,
    p.special_price_end_at,
    p.is_active,
    p.created_at,
    p.updated_at,
    p.cost_price,
    p.product_code,
    p.description,
    p.storage_location
FROM products p
LEFT JOIN product_types pt ON pt.id = p.product_type_id
LEFT JOIN product_units pu ON pu.id = p.product_unit_id
LEFT JOIN product_brands pb ON pb.id = p.brand_id
LEFT JOIN stocks s ON s.product_id = p.id
GROUP BY
    p.id, p.store_id, p.product_type_id, pt.name,
    p.product_unit_id, pu.name, p.brand_id, pb.name,
    p.name, p.sku, p.barcode, p.image_url,
    p.min_stock, p.max_stock,
    p.base_price, p.special_price,
    p.special_price_start_at, p.special_price_end_at,
    p.is_active, p.created_at, p.updated_at,
    p.cost_price, p.product_code, p.description, p.storage_location;
