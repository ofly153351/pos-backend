-- 038_stock_adjustment_reason_idempotency.sql — Phase W2: canonical quick stock
-- adjustments. Adds the audit + safety columns the adjustment workflow needs:
--
--   * stock_movements.reason             — the stable adjustment REASON CODE
--                                          (FOUND_EXTRA / DAMAGED / SPOT_COUNT / OTHER …),
--                                          stored alongside the free-text note.
--   * stock_movements.idempotency_key    — a client-supplied key making a duplicate
--                                          submit (double-click / retry) a no-op.
--   * stock_movements.request_fingerprint — a deterministic hash of the request's
--                                          business-significant fields (store, product,
--                                          location, mode, requested quantity, reason,
--                                          normalized note, user). Reusing a key with the
--                                          SAME fingerprint is a safe retry (returns the
--                                          original); reusing it with a DIFFERENT
--                                          fingerprint is a conflict (HTTP 409). It hashes
--                                          the REQUESTED quantity (e.g. SET_ACTUAL's
--                                          physical qty), never the live-stock-dependent
--                                          computed delta, so it captures user intent.
--
-- A DEDICATED idempotency_key column is used rather than reusing reference_id: a single
-- sale already writes one SALE movement per line all sharing the sale's reference_id, so
-- a UNIQUE index on reference_id could never be created on existing data. idempotency_key
-- is only set by the adjustment workflow, so the partial UNIQUE index below is safe.
--
-- Safety / scope:
--   * Additive columns (nullable, no default backfill). No stock quantities, movement
--     history, product defaults or warehouse_inventory are read or written here.
--   * The partial UNIQUE index only constrains rows where idempotency_key IS NOT NULL;
--     all existing rows have NULL, so it builds with zero possibility of a duplicate-key
--     failure.
--   * Runs once via the schema_migrations ledger, inside one transaction. Every
--     statement is idempotent (IF NOT EXISTS). Forward-only (no down file).

ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS reason              TEXT;
ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS idempotency_key     TEXT;
ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS request_fingerprint TEXT;

-- One adjustment per (store, idempotency_key): a retried submit collides here and is
-- resolved to the original movement instead of double-applying the stock change.
CREATE UNIQUE INDEX IF NOT EXISTS stock_movements_store_idempotency_key
    ON stock_movements (store_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
