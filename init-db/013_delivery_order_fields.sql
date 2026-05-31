-- Delivery order specific fields on documents
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS delivery_date      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS delivery_address   TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS delivery_contact   TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS delivery_phone     TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sales_zone         TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS salesperson_name   TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS invoice_ref_no     TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS po_ref_no          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS shipping_fee       NUMERIC(14,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS credit_term_days   INTEGER     NOT NULL DEFAULT 0;
