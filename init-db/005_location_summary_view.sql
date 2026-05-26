-- ── Location Summary View ────────────────────────────────────
-- Combines locations with warehouse info, live stock stats,
-- and a computed status field for reporting / admin queries.
CREATE OR REPLACE VIEW location_summary AS
SELECT
    l.id,
    l.store_id,
    l.warehouse_id,
    COALESCE(w.name, '')    AS warehouse_name,
    COALESCE(w.code, '')    AS warehouse_code,
    l.zone_name,
    l.floor_name,
    l.code,
    l.name,
    l.is_sale_point,
    l.is_active,
    COALESCE(SUM(s.quantity), 0)::INTEGER                                          AS total_stock,
    COUNT(DISTINCT CASE WHEN s.quantity > 0 THEN s.product_id END)::INTEGER        AS product_count,
    CASE
        WHEN NOT l.is_active  THEN 'inactive'
        WHEN l.is_sale_point  THEN 'sale_point'
        ELSE                       'storage'
    END                     AS computed_status,
    l.created_at,
    l.updated_at
FROM locations l
LEFT JOIN warehouses w ON w.id = l.warehouse_id
LEFT JOIN stocks     s ON s.location_id = l.id
GROUP BY
    l.id, l.store_id, l.warehouse_id,
    w.name, w.code,
    l.zone_name, l.floor_name, l.code, l.name,
    l.is_sale_point, l.is_active,
    l.created_at, l.updated_at;
