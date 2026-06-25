-- 047_warehouses_locations_deleted_at.sql
-- Soft-delete (Archive) markers for warehouses and storage locations.
--
-- This is DISTINCT from is_active: is_active is the deactivate/reactivate status that
-- intentionally keeps a warehouse/location visible (and reactivatable) on its management
-- page. deleted_at marks the record as ARCHIVED — hidden from every active management view
-- and operational selector (receiving, PO warehouse, stock adjust/count/transfer, product
-- default-location picker, warehouse inventory) — while its row and ALL historical
-- references (stocks, stock_movements, warehouse_receipts, warehouse_inventory, stock
-- count sessions, documents, reports, audit logs) are preserved. Archiving never destroys
-- operational data and never zeroes stock.
--
-- Mirrors the products soft-delete precedent (046): the Go structs deliberately do NOT map
-- a DeletedAt field, so historical LEFT JOINs to warehouses/locations keep resolving
-- archived names automatically. Only active/operational list queries filter
-- `deleted_at IS NULL`.
--
-- Additive + idempotent (nullable column, IF NOT EXISTS); no FK or constraint changes; no
-- cascade-delete relationships are introduced.
ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE locations  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Partial indexes keep the common "active (non-archived) records in a store" lookups fast;
-- archived rows are excluded from the index entirely.
CREATE INDEX IF NOT EXISTS idx_warehouses_store_not_deleted
    ON warehouses (store_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_locations_store_not_deleted
    ON locations (store_id)
    WHERE deleted_at IS NULL;
