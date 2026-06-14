-- 039_receipt_confirm_idempotency.sql — Phase W3: Goods Receiving confirmation
-- idempotency. A receipt's status guard alone cannot make a network-retried confirm
-- safe (the retry would error instead of returning the original result). These columns
-- let an identical retry (same key + same request fingerprint) return the original
-- confirmed receipt, while a reused key with a DIFFERENT fingerprint is a conflict.
--
--   * warehouse_receipts.confirm_idempotency_key     — client-supplied key for the
--                                                      confirm request.
--   * warehouse_receipts.confirm_request_fingerprint — deterministic hash of the
--                                                      confirm request's business fields
--                                                      (store, receipt, pre-status,
--                                                      confirming user, ordered line IDs +
--                                                      resolved locations + quantities +
--                                                      unit costs). Never includes
--                                                      computed post-confirmation values.
--
-- Safety / scope:
--   * Additive nullable columns; no receipt/stock/movement/cost/PO data is read or
--     written here. Existing receipts get NULL (un-keyed) and are unaffected.
--   * The partial UNIQUE index only constrains rows where confirm_idempotency_key IS NOT
--     NULL; all existing rows have NULL, so it builds with no possibility of a
--     duplicate-key failure. Scope is per-store + per-key (cross-store keys never clash).
--   * Runs once via the schema_migrations ledger, inside one transaction. Every
--     statement is idempotent (IF NOT EXISTS). Forward-only (no down file).

ALTER TABLE warehouse_receipts ADD COLUMN IF NOT EXISTS confirm_idempotency_key     TEXT;
ALTER TABLE warehouse_receipts ADD COLUMN IF NOT EXISTS confirm_request_fingerprint TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS warehouse_receipts_store_confirm_idempotency_key
    ON warehouse_receipts (store_id, confirm_idempotency_key) WHERE confirm_idempotency_key IS NOT NULL;
