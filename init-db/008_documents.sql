-- Document management tables

CREATE TABLE IF NOT EXISTS documents (
    id               VARCHAR(30)  PRIMARY KEY,
    store_id         VARCHAR(30)  NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    document_no      VARCHAR(40)  NOT NULL UNIQUE,
    document_no_full VARCHAR(60)  NOT NULL,
    type             VARCHAR(20)  NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    payment_status   VARCHAR(20)  NOT NULL DEFAULT 'UNPAID',
    customer_id      VARCHAR(30)  NOT NULL,
    customer_name    TEXT         NOT NULL,
    customer_tax_id  TEXT,
    staff_id         VARCHAR(30)  NOT NULL,
    staff_name       TEXT         NOT NULL,
    document_date    TIMESTAMPTZ  NOT NULL,
    due_date         TIMESTAMPTZ,
    subtotal         NUMERIC(14,2) NOT NULL DEFAULT 0,
    vat_rate         NUMERIC(5,2)  NOT NULL DEFAULT 0,
    vat_amount       NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_amount     NUMERIC(14,2) NOT NULL DEFAULT 0,
    notes            TEXT,
    created_by       VARCHAR(30)  NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_documents_store_id      ON documents(store_id);
CREATE INDEX IF NOT EXISTS idx_documents_type          ON documents(store_id, type);
CREATE INDEX IF NOT EXISTS idx_documents_status        ON documents(store_id, status);
CREATE INDEX IF NOT EXISTS idx_documents_payment_status ON documents(store_id, payment_status);
CREATE INDEX IF NOT EXISTS idx_documents_document_date ON documents(store_id, document_date DESC);

CREATE TABLE IF NOT EXISTS document_items (
    id             VARCHAR(30)   PRIMARY KEY,
    document_id    VARCHAR(30)   NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    product_id     VARCHAR(30),
    description    TEXT          NOT NULL,
    quantity       NUMERIC(14,4) NOT NULL DEFAULT 1,
    unit_price     NUMERIC(14,2) NOT NULL DEFAULT 0,
    discount_type  VARCHAR(10)   NOT NULL DEFAULT '',
    discount_value NUMERIC(14,2) NOT NULL DEFAULT 0,
    amount         NUMERIC(14,2) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_document_items_document_id ON document_items(document_id);
