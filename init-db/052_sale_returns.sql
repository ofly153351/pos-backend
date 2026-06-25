-- 052_sale_returns.sql
-- Partial-return architecture. A return is a SEPARATE record linked to a sale;
-- the original sale row is never voided. Returning some-or-all items moves the
-- sale to 'partially_returned' / 'fully_returned' (distinct from 'voided', which
-- cancels the entire bill). Refund/stock-restore is tracked per return.

CREATE TABLE IF NOT EXISTS sale_returns (
    id            TEXT PRIMARY KEY,
    store_id      TEXT NOT NULL,
    sale_id       TEXT NOT NULL,
    return_number TEXT NOT NULL,
    refund_method TEXT NOT NULL,            -- cash | transfer | card | qr
    refund_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    reason        TEXT,
    created_by    TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sale_returns_sale  ON sale_returns (sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_returns_store ON sale_returns (store_id, created_at DESC);

CREATE TABLE IF NOT EXISTS sale_return_items (
    id           TEXT PRIMARY KEY,
    return_id    TEXT NOT NULL REFERENCES sale_returns(id) ON DELETE CASCADE,
    sale_item_id TEXT NOT NULL,             -- original sale_items.id
    product_id   TEXT NOT NULL,
    product_name TEXT,
    sku          TEXT,
    quantity     INTEGER NOT NULL,
    unit_price   NUMERIC(14,2) NOT NULL DEFAULT 0,  -- effective per-unit paid (line_total/qty)
    line_refund  NUMERIC(14,2) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sale_return_items_return ON sale_return_items (return_id);

-- Running returned tally per original line so remaining = quantity - returned_quantity.
ALTER TABLE sale_items ADD COLUMN IF NOT EXISTS returned_quantity INTEGER NOT NULL DEFAULT 0;

-- sales.status allowed values now:
--   completed | partially_returned | fully_returned | voided
-- Migration 042 added a CHECK limited to completed/voided — widen it.
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_status_check;
ALTER TABLE sales ADD CONSTRAINT sales_status_check
    CHECK (status = ANY (ARRAY['completed', 'partially_returned', 'fully_returned', 'voided']));
