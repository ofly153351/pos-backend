CREATE TABLE IF NOT EXISTS customer_shipping_addresses (
    id                   VARCHAR(30)  PRIMARY KEY,
    customer_id          VARCHAR(30)  NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    label                VARCHAR(100) NOT NULL DEFAULT '',
    recipient_name       VARCHAR(200) NOT NULL DEFAULT '',
    recipient_phone      VARCHAR(50)  NOT NULL DEFAULT '',
    address              TEXT         NOT NULL DEFAULT '',
    sub_district         VARCHAR(100) NOT NULL DEFAULT '',
    district             VARCHAR(100) NOT NULL DEFAULT '',
    province             VARCHAR(100) NOT NULL DEFAULT '',
    postal_code          VARCHAR(20)  NOT NULL DEFAULT '',
    note                 TEXT         NOT NULL DEFAULT '',
    use_customer_address BOOLEAN      NOT NULL DEFAULT FALSE,
    is_default           BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Idempotent backfill: a DB whose table was created before sub_district was added
-- skips the CREATE TABLE above (IF NOT EXISTS) and would otherwise lack the column.
ALTER TABLE customer_shipping_addresses
    ADD COLUMN IF NOT EXISTS sub_district VARCHAR(100) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_csa_customer_id ON customer_shipping_addresses(customer_id);
