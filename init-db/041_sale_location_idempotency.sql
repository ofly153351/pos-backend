-- Phase W4B — location-aware POS sale deduction.
-- A sale now records the single effective sale-point Location it deducted from (one
-- Location per sale in W4B). Nullable so historical sales (pre-W4B) stay accurate rather
-- than being backfilled to an inaccurate current default.
ALTER TABLE sales ADD COLUMN IF NOT EXISTS location_id TEXT;

-- Request-level idempotency: a retried sale-create with the same key returns the original
-- sale instead of creating a second sale / second payment / second stock deduction.
ALTER TABLE sales ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS request_fingerprint TEXT;

-- Database-level idempotency uniqueness, scoped per store; partial so non-idempotent
-- (key-less) sales are unaffected.
CREATE UNIQUE INDEX IF NOT EXISTS sales_store_idempotency_key
    ON sales (store_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
