-- 027: Promotion auditability — soft-delete + persisted analytics.
--
--  deleted_at:            hide a promotion from active lists / checkout verification
--                         without losing historical references (sales/usage rows).
--  usage_count:           number of sales that applied the promotion.
--  discount_given_total:  cumulative discount attributed to the promotion.
--
-- usage_count / discount_given_total replace the always-0 client placeholders and
-- are updated inside the sale-create transaction (see sale.recordPromotionUsage).
--
-- Idempotent + additive (ADD COLUMN IF NOT EXISTS); safe for existing databases.
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS deleted_at TEXT;
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS usage_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS discount_given_total NUMERIC(14,2) NOT NULL DEFAULT 0;
