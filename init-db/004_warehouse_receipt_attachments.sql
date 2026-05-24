CREATE TABLE IF NOT EXISTS warehouse_receipt_attachments (
    id          TEXT        PRIMARY KEY,
    receipt_id   TEXT       NOT NULL REFERENCES warehouse_receipts(id) ON DELETE CASCADE,
    url          TEXT       NOT NULL,
    mime_type    TEXT       NOT NULL,
    name         TEXT       NOT NULL,
    size         BIGINT     NOT NULL DEFAULT 0 CHECK (size >= 0),
    uploaded_by  TEXT       NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_warehouse_receipt_attachments_receipt_id ON warehouse_receipt_attachments(receipt_id);

CREATE TABLE IF NOT EXISTS warehouse_receipt_pending_attachments (
    id          TEXT        PRIMARY KEY,
    receipt_id   TEXT       NOT NULL REFERENCES warehouse_receipts(id) ON DELETE CASCADE,
    mime_type    TEXT       NOT NULL,
    name         TEXT       NOT NULL,
    size         BIGINT     NOT NULL DEFAULT 0 CHECK (size >= 0),
    data         BYTEA      NOT NULL,
    created_by   TEXT       NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_warehouse_receipt_pending_attachments_receipt_id ON warehouse_receipt_pending_attachments(receipt_id);
