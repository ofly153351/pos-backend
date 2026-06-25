-- 050_customers_shipping.sql
-- Phase 3: customer shipping / delivery profile. Delivery Orders auto-fill their
-- delivery contact / phone / address from these fields (with per-document override).
-- Separate from the billing `address` (029_customers_tax_branch). Nullable + IF NOT
-- EXISTS so the customer repo's empty->NULL idiom round-trips cleanly, existing rows
-- are untouched, and the migration is idempotent / re-runnable.

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS shipping_contact     VARCHAR(120),
    ADD COLUMN IF NOT EXISTS shipping_phone       VARCHAR(30),
    ADD COLUMN IF NOT EXISTS shipping_address     TEXT,
    ADD COLUMN IF NOT EXISTS shipping_province    VARCHAR(80),
    ADD COLUMN IF NOT EXISTS shipping_district    VARCHAR(80),
    ADD COLUMN IF NOT EXISTS shipping_postal_code VARCHAR(10),
    ADD COLUMN IF NOT EXISTS delivery_note        TEXT;
