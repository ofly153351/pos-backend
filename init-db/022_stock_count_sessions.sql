-- 022_stock_count_sessions.sql — Stock count (physical inventory) persistence.
-- Sessions and their per-product count items live server-side so a count can be
-- saved as a draft, continued later, and reviewed/approved from another terminal.
-- The applied quantity corrections still flow through stock_movements
-- (COUNT_CORRECTION) — this only persists the session/worksheet state.

CREATE TABLE IF NOT EXISTS stock_count_sessions (
    id             TEXT        PRIMARY KEY,
    store_id       TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name           TEXT        NOT NULL DEFAULT '',
    warehouse_name TEXT,
    zone           TEXT,
    category_id    TEXT,
    category_name  TEXT,
    staff          TEXT        NOT NULL DEFAULT '',
    note           TEXT        NOT NULL DEFAULT '',
    status         TEXT        NOT NULL DEFAULT 'draft',
    count_type     TEXT        NOT NULL DEFAULT 'full',
    cycle_rule     TEXT        NOT NULL DEFAULT '',
    blind_count    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by     TEXT        NOT NULL DEFAULT '',
    completed_by   TEXT        NOT NULL DEFAULT '',
    completed_at   TEXT,
    created_at     TEXT        NOT NULL DEFAULT '',
    updated_at     TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_stock_count_sessions_store ON stock_count_sessions(store_id, created_at DESC);

CREATE TABLE IF NOT EXISTS stock_count_items (
    id                    TEXT          PRIMARY KEY,
    session_id            TEXT          NOT NULL REFERENCES stock_count_sessions(id) ON DELETE CASCADE,
    product_id            TEXT          NOT NULL DEFAULT '',
    name                  TEXT          NOT NULL DEFAULT '',
    sku                   TEXT          NOT NULL DEFAULT '',
    barcode               TEXT          NOT NULL DEFAULT '',
    system_qty            INTEGER       NOT NULL DEFAULT 0,
    min_stock             INTEGER       NOT NULL DEFAULT 0,
    location              TEXT          NOT NULL DEFAULT '',
    counted               INTEGER,
    note                  TEXT          NOT NULL DEFAULT '',
    skipped               BOOLEAN       NOT NULL DEFAULT FALSE,
    variance_reason       TEXT          NOT NULL DEFAULT '',
    variance_reason_other TEXT          NOT NULL DEFAULT '',
    count_user            TEXT          NOT NULL DEFAULT '',
    counted_at            TEXT,
    cost_basis            NUMERIC(14,2) NOT NULL DEFAULT 0,
    adjusted              BOOLEAN       NOT NULL DEFAULT FALSE,
    adjusted_at           TEXT,
    adjusted_by           TEXT          NOT NULL DEFAULT '',
    sort_order            INTEGER       NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_stock_count_items_session ON stock_count_items(session_id, sort_order);
