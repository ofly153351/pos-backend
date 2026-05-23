ALTER TABLE locations
    ADD COLUMN IF NOT EXISTS zone_name TEXT,
    ADD COLUMN IF NOT EXISTS floor_name TEXT;

CREATE TABLE IF NOT EXISTS warehouse_receipt_document_sequences (
    date_key      TEXT        PRIMARY KEY,
    last_sequence INTEGER     NOT NULL DEFAULT 0 CHECK (last_sequence >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS warehouse_receipts (
    id                   TEXT          PRIMARY KEY,
    store_id             TEXT          NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    warehouse_id         TEXT          NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    supplier_id          TEXT          REFERENCES suppliers(id) ON DELETE SET NULL,
    purchase_order_id    TEXT          REFERENCES purchase_orders(id) ON DELETE SET NULL,
    document_no          TEXT          NOT NULL UNIQUE,
    status               TEXT          NOT NULL CHECK (status IN ('draft','confirmed','cancelled')),
    received_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    reference_no         TEXT,
    note                 TEXT,
    vat_included         BOOLEAN       NOT NULL DEFAULT TRUE,
    vat_percent          NUMERIC(5,2)  NOT NULL DEFAULT 7 CHECK (vat_percent >= 0 AND vat_percent <= 100),
    total_items          INTEGER       NOT NULL DEFAULT 0 CHECK (total_items >= 0),
    subtotal_amount      NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (subtotal_amount >= 0),
    discount_amount      NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    net_amount           NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (net_amount >= 0),
    vat_amount           NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (vat_amount >= 0),
    total_amount         NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    attachment_url       TEXT,
    attachment_mime_type TEXT,
    attachment_name      TEXT,
    attachment_size      BIGINT        NOT NULL DEFAULT 0 CHECK (attachment_size >= 0),
    created_by           TEXT          NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    confirmed_by         TEXT          REFERENCES users(id) ON DELETE SET NULL,
    cancelled_by         TEXT          REFERENCES users(id) ON DELETE SET NULL,
    confirmed_at         TIMESTAMPTZ,
    cancelled_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_warehouse_receipts_store_id ON warehouse_receipts(store_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_receipts_warehouse_id ON warehouse_receipts(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_receipts_status ON warehouse_receipts(status);
CREATE INDEX IF NOT EXISTS idx_warehouse_receipts_purchase_order_id ON warehouse_receipts(purchase_order_id);

CREATE TABLE IF NOT EXISTS warehouse_receipt_items (
    id              TEXT          PRIMARY KEY,
    receipt_id       TEXT         NOT NULL REFERENCES warehouse_receipts(id) ON DELETE CASCADE,
    product_id       TEXT         NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    location_id      TEXT         NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
    warehouse_id     TEXT         NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    zone_name        TEXT,
    floor_name       TEXT,
    location_name    TEXT         NOT NULL,
    product_name     TEXT         NOT NULL,
    sku              TEXT,
    barcode          TEXT,
    unit_name        TEXT,
    quantity         INTEGER      NOT NULL CHECK (quantity >= 1),
    unit_price       NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    discount_type    TEXT,
    discount_value   NUMERIC(12,4),
    discount_amount  NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    line_subtotal    NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (line_subtotal >= 0),
    line_net         NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (line_net >= 0),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (receipt_id, product_id, location_id)
);

CREATE INDEX IF NOT EXISTS idx_warehouse_receipt_items_receipt_id ON warehouse_receipt_items(receipt_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_receipt_items_product_id ON warehouse_receipt_items(product_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_receipt_items_location_id ON warehouse_receipt_items(location_id);

CREATE TABLE IF NOT EXISTS warehouse_receipt_audits (
    id          TEXT        PRIMARY KEY,
    receipt_id   TEXT       NOT NULL REFERENCES warehouse_receipts(id) ON DELETE CASCADE,
    action       TEXT       NOT NULL,
    description  TEXT,
    actor_id     TEXT       NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_warehouse_receipt_audits_receipt_id ON warehouse_receipt_audits(receipt_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_receipt_audits_actor_id ON warehouse_receipt_audits(actor_id);
