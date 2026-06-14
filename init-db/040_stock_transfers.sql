-- Phase W4A — canonical location-aware stock transfer.
-- One header row per successful transfer: the audit/document for a Product moving between
-- two explicit Locations in the same Store. The paired TRANSFER_OUT/TRANSFER_IN rows in
-- stock_movements reference this row via reference_id (shared transfer reference).
CREATE TABLE IF NOT EXISTS stock_transfers (
    id                   TEXT PRIMARY KEY,
    store_id             TEXT        NOT NULL,
    product_id           TEXT        NOT NULL,
    source_location_id   TEXT        NOT NULL,
    dest_location_id     TEXT        NOT NULL,
    quantity             INTEGER     NOT NULL,
    reason               TEXT        NOT NULL,
    note                 TEXT        NOT NULL DEFAULT '',
    created_by           TEXT        NOT NULL,
    idempotency_key      TEXT,
    request_fingerprint  TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Database-level idempotency: a (store, key) may exist at most once, so a retried transfer
-- that races past the service pre-check is rejected by this unique index (Phase W4A §9).
CREATE UNIQUE INDEX IF NOT EXISTS stock_transfers_store_idempotency_key
    ON stock_transfers (store_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- History lookups by product.
CREATE INDEX IF NOT EXISTS stock_transfers_store_product
    ON stock_transfers (store_id, product_id);
