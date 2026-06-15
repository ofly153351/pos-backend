-- 043_product_stock_aggregate_contract.sql
-- Phase W5 — clarify the product stock aggregate contract.
--
-- The historical product_view (001/034) named its two aggregates MISLEADINGLY:
--   total_stock     = SUM(quantity) WHERE location.is_sale_point = TRUE   (actually the
--                     POS-sellable "ready" subset, NOT the grand total)
--   warehouse_stock = SUM(quantity) across EVERY location                 (actually the
--                     grand total, NOT a warehouse-only figure)
-- This inversion already forced the Transfer drawer to hand-map field meanings (W4A).
--
-- W5 is ADDITIVE and NON-BREAKING: it appends two correctly-named aggregates so new and
-- migrated consumers can use unambiguous fields, WITHOUT changing the meaning of the two
-- legacy columns (which active, partly concurrent-owned consumers still read):
--   ready_stock   = SUM(quantity) WHERE is_sale_point = TRUE   (POS-sellable; == legacy total_stock)
--   storage_stock = SUM(quantity) WHERE is_sale_point = FALSE  (non-sale-point storage)
-- Invariant: ready_stock + storage_stock == legacy warehouse_stock (the grand total).
-- Stock in INACTIVE locations is still counted in all aggregates (only NEW operational
-- selection excludes inactive locations); this preserves existing totals exactly.
--
-- CREATE OR REPLACE VIEW can only APPEND columns, so the existing column list/order is
-- reproduced verbatim from migration 034 and the two new aggregates are added at the end.
-- Idempotent (CREATE OR REPLACE). No data is written or rewritten — a view is a query.

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
    -- Phase W5 — correctly-named aggregates (appended):
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
    ), 0)::integer AS storage_stock
FROM products p
LEFT JOIN product_types  pt ON pt.id = p.product_type_id
LEFT JOIN product_units  pu ON pu.id = p.product_unit_id
LEFT JOIN product_brands pb ON pb.id = p.brand_id;
