-- 036_restrict_warehouse_location_stock_deletes.sql — Phase W0.5: Database
-- Referential Safety.
--
-- The core Warehouse → Location → Stock hierarchy previously cascaded deletes:
--   locations.warehouse_id → warehouses(id)  ON DELETE CASCADE
--   stocks.location_id      → locations(id)   ON DELETE CASCADE
-- meaning a single DELETE of a warehouse or location would silently destroy every
-- dependent location, stock row and (via stock_movements SET NULL) blur the audit.
-- Phase W0 added application guards (warehouseHasReferences / locationInUse → HTTP
-- 409) but the database itself was still willing to cascade. This migration makes
-- the DATABASE the final backstop by switching both foreign keys to ON DELETE
-- RESTRICT, so Postgres refuses to delete a parent that still has dependent rows.
--
-- Layered behaviour after this migration:
--   Frontend disabled/warned  →  App guard returns friendly 409  →  DB RESTRICT
--
-- Notes / safety:
--   * RESTRICT is chosen to match the project convention already used by
--     warehouse_receipts / warehouse_receipt_items (both ON DELETE RESTRICT) and to
--     reject dependent-parent deletes immediately and explicitly. For non-deferrable
--     single-row deletes RESTRICT and NO ACTION behave the same; RESTRICT is the
--     more explicit, project-consistent choice.
--   * Only the ON DELETE action changes. Columns, parent tables, NOT NULL
--     nullability and the default ON UPDATE (NO ACTION) are unchanged.
--   * Dropping a FK constraint does NOT drop its supporting index, so
--     idx_locations_warehouse_id, idx_stocks_location_id and the unique keys
--     (locations_store_id_warehouse_id_code_key, stocks_product_id_location_id_key)
--     are preserved untouched.
--   * No application data is read, written, reset or rebuilt. Verified zero orphan
--     rows exist, so re-adding the constraint validates cleanly.
--   * Runs exactly once via the schema_migrations ledger, inside one transaction
--     (atomic). DROP ... IF EXISTS keeps it safe if ever re-applied.
--   * Forward-only project: there is no down/rollback file. Reverting to CASCADE
--     would REINTRODUCE the destructive behaviour and must never be done casually.
--   * NOT changed here (documented follow-up): warehouse_inventory.warehouse_id
--     (ON DELETE CASCADE) and stocks.product_id are out of W0.5 scope; the W0
--     warehouse/location guards already cover the reachable delete paths.

-- locations.warehouse_id → warehouses(id): CASCADE → RESTRICT
ALTER TABLE locations DROP CONSTRAINT IF EXISTS locations_warehouse_id_fkey;
ALTER TABLE locations
    ADD CONSTRAINT locations_warehouse_id_fkey
    FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE RESTRICT;

-- stocks.location_id → locations(id): CASCADE → RESTRICT
ALTER TABLE stocks DROP CONSTRAINT IF EXISTS stocks_location_id_fkey;
ALTER TABLE stocks
    ADD CONSTRAINT stocks_location_id_fkey
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE RESTRICT;
