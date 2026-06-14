-- 037_default_warehouse_sale_location.sql — Phase W1: Automatic Default Warehouse
-- and "หน้าร้าน" Sale Location — authoritative default identity.
--
-- Goal: give every store a stable, rename-proof way to identify its ONE default
-- warehouse ("คลังหลัก") and its ONE default sale-point location ("หน้าร้าน"), so a
-- small shop can start selling without hand-configuring warehouse/location records.
--
-- MODEL CHOICE — Option B (boolean flags), NOT Option A (stores.default_* FK refs):
--   warehouses.store_id and locations.store_id are already ON DELETE CASCADE. Adding
--   stores.default_warehouse_id -> warehouses(id) (Option A) would introduce a
--   CIRCULAR foreign key (stores <-> warehouses). Combined with that existing cascade
--   and the Phase W0.5 RESTRICT policy, a store delete would trip its own cascade
--   against a RESTRICT-ing self-reference — a genuinely problematic circular-FK
--   behaviour. Option B adds NO new foreign key: the default marker lives on the
--   warehouse/location row itself (already scoped by its own store_id, so a default
--   can never point at the wrong store), survives renames (it is a flag, not a name),
--   and is enforced at the database level by a partial UNIQUE index.
--
-- What this migration does (schema/metadata only — NO data movement):
--   * Adds warehouses.is_default       (BOOLEAN NOT NULL DEFAULT FALSE)
--   * Adds locations.is_default_sale   (BOOLEAN NOT NULL DEFAULT FALSE)
--   * Adds a PARTIAL UNIQUE index per table enforcing "at most one default per store".
--
-- Safety / scope:
--   * No stock rows, stock_movements, quantities, product default_location_id, or
--     warehouse_inventory are read or written here. Defaults DEFAULT to FALSE, so no
--     existing store is marked yet — the canonical provisioning service (new-store
--     creation + idempotent startup backfill) sets the flags deterministically.
--   * Existing warehouse/location foreign keys (incl. the W0.5 RESTRICT ones) are
--     untouched. Columns are additive; existing rows are unaffected.
--   * Runs exactly once via the schema_migrations ledger, inside one transaction.
--     Every statement is idempotent (IF NOT EXISTS) so a re-run is harmless.
--   * Forward-only project: no down file. Dropping these columns later would only
--     lose the default markers; it would not destroy any operational data.

ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS is_default      BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE locations  ADD COLUMN IF NOT EXISTS is_default_sale BOOLEAN NOT NULL DEFAULT FALSE;

-- At most one default warehouse per store (partial: only constrains the flagged rows).
CREATE UNIQUE INDEX IF NOT EXISTS warehouses_one_default_per_store
    ON warehouses (store_id) WHERE is_default;

-- At most one default sale location per store.
CREATE UNIQUE INDEX IF NOT EXISTS locations_one_default_sale_per_store
    ON locations (store_id) WHERE is_default_sale;
