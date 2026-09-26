-- Receipt Type 2 settlement references. One receipt can settle many
-- billing notices and delivery orders without encoding references in text.
CREATE TABLE IF NOT EXISTS receipt_settlements (
    id                   VARCHAR(30) PRIMARY KEY,
    receipt_document_id  VARCHAR(30) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    billing_document_id  VARCHAR(30) REFERENCES documents(id) ON DELETE RESTRICT,
    delivery_order_id    VARCHAR(30) REFERENCES documents(id) ON DELETE RESTRICT,
    applied_amount       NUMERIC(14,2) NOT NULL,
    sort_order           INTEGER NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT receipt_settlements_source_check CHECK (
        billing_document_id IS NOT NULL OR delivery_order_id IS NOT NULL
    ),
    CONSTRAINT receipt_settlements_amount_check CHECK (applied_amount > 0),
    CONSTRAINT receipt_settlements_unique_source UNIQUE (
        receipt_document_id, billing_document_id, delivery_order_id
    )
);

CREATE INDEX IF NOT EXISTS idx_receipt_settlements_receipt
    ON receipt_settlements(receipt_document_id, sort_order);
