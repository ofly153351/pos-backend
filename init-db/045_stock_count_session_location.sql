-- 045: Location-scoped Stock Count.
--
-- A count session targets exactly ONE storage location. The column is NULLABLE so that
-- historical sessions (created before this migration) remain readable; the Apply path
-- rejects a NULL-location session, because a count whose system quantity is a store/warehouse
-- aggregate can never be safely written into a single location. New sessions set location_id
-- to a valid active location, and every count item's expected quantity is that location's
-- on-hand. ON DELETE RESTRICT mirrors stocks.location_id / warehouse_receipt_items.location_id
-- (a location that has been counted cannot be silently deleted).
ALTER TABLE stock_count_sessions
    ADD COLUMN IF NOT EXISTS location_id TEXT REFERENCES locations(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_stock_count_sessions_location
    ON stock_count_sessions (location_id);
