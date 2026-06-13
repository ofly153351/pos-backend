-- 032_purchase_order_audit.sql
-- P2.5 audit trail: record who created a purchase order and who received stock
-- against it. created_at / received_at already exist (001_schema.sql); this adds
-- the actor ids. ON DELETE SET NULL so removing a user never blocks PO history.
-- Nullable (existing rows have no recorded actor). Idempotent via IF NOT EXISTS.

ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS created_by  TEXT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS received_by TEXT REFERENCES users(id) ON DELETE SET NULL;
