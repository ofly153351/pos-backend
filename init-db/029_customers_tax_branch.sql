-- 029_customers_tax_branch.sql
-- P1.1: persist customer tax_id (Thai 13-digit) + branch (short label).
-- The frontend already sends tax_id/branch end-to-end; the backend struct +
-- repo additions accompany this migration. Nullable (no NOT NULL/default) so the
-- customer repo's empty->NULL idiom round-trips cleanly and existing rows are
-- untouched. IF NOT EXISTS keeps it idempotent / re-runnable.

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS tax_id VARCHAR(13),
    ADD COLUMN IF NOT EXISTS branch VARCHAR(50);
