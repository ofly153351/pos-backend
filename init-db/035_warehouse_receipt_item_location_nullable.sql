-- 035_warehouse_receipt_item_location_nullable.sql
-- Phase 2 (auto receiving location): a receipt item's location is now resolved
-- from products.default_location_id instead of being chosen manually. A product
-- that has no default location yet may still be added to a DRAFT for correction,
-- so a draft item's location_id can be temporarily NULL. Submit/Confirm enforce a
-- valid non-null location, and confirmed items always carry a real location.
--
-- Additive / non-destructive: existing rows already have a location_id, so
-- relaxing the NOT NULL constraint changes nothing for them. Idempotent — running
-- DROP NOT NULL again on an already-nullable column is a no-op. No backfill.

ALTER TABLE warehouse_receipt_items ALTER COLUMN location_id DROP NOT NULL;
