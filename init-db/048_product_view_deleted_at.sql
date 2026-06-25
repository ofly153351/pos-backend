-- 048_product_view_deleted_at.sql
-- Expose products.deleted_at through product_view so every inventory/KPI consumer of the
-- view can exclude soft-deleted products with a simple `deleted_at IS NULL` predicate.
--
-- Migration 046 added products.deleted_at (the soft-delete marker, distinct from is_active).
-- product/repository.go already joins products to filter deleted rows, but the many OTHER
-- readers of product_view (warehouse inventory, dashboards, finance reports, pickers) cannot
-- add a plain `products` join without column-name clashes (cost_price/min_stock exist in both
-- products and the view). Surfacing deleted_at on the view itself gives all of them an
-- unambiguous, index-friendly filter.
--
-- (Numbered 048 because a concurrent migration already claims 047.)
--
-- ADDITIVE + NON-BREAKING: CREATE OR REPLACE VIEW may only APPEND columns, so the column
-- list/order from migration 043 is reproduced verbatim and deleted_at is appended at the end.
-- Idempotent (CREATE OR REPLACE). No data is written or rewritten — a view is a query.
-- The view still includes deleted products (so historical/PO resolution can still read them);
-- only operational/KPI queries opt in to `deleted_at IS NULL`.

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
    p.default_location_id,
    -- Phase W5 — correctly-named aggregates:
    COALESCE((
        SELECT SUM(s.quantity)
        FROM stocks s
        JOIN locations l ON l.id = s.location_id AND l.is_sale_point = TRUE
        WHERE s.product_id = p.id
    ), 0)::integer AS ready_stock,
    COALESCE((
        SELECT SUM(s.quantity)
        FROM stocks s
        JOIN locations l ON l.id = s.location_id AND l.is_sale_point = FALSE
        WHERE s.product_id = p.id
    ), 0)::integer AS storage_stock,
    -- 048 — soft-delete marker (appended; non-breaking):
    p.deleted_at AS deleted_at
FROM products p
LEFT JOIN product_types  pt ON pt.id = p.product_type_id
LEFT JOIN product_units  pu ON pu.id = p.product_unit_id
LEFT JOIN product_brands pb ON pb.id = p.brand_id;
