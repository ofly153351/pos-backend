-- 024_promotions.sql — Promotions / campaigns (server-persisted, store-scoped).
-- The full campaign object (11 types, ~45 mostly type-specific fields) is stored
-- verbatim in `data` so the rich client shape round-trips without per-field columns;
-- a few fields are mirrored into columns for listing/filtering. Evaluation at
-- checkout is client-side in this phase (see promotion-engine.ts).

CREATE TABLE IF NOT EXISTS promotions (
    id         TEXT PRIMARY KEY,
    store_id   TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name       TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'draft',
    code       TEXT NOT NULL DEFAULT '',
    data       TEXT NOT NULL DEFAULT '{}',
    created_by TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_promotions_store ON promotions(store_id, created_at DESC);
