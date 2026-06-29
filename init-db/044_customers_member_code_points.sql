-- 044_customers_member_code_points.sql
-- The customers UI shows a member code, loyalty points, total purchase value and
-- bill count. Only member_code + points are persisted columns; total_purchase /
-- total_bills are computed at read time from the sales table (see customer repo
-- ListByStore). member_code is auto-generated per store ("M00001", ...) and shown
-- read-only in the form, so it is nullable here and filled by the backend on create.
-- IF NOT EXISTS keeps this idempotent / re-runnable.

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS member_code TEXT,
    ADD COLUMN IF NOT EXISTS points      INTEGER NOT NULL DEFAULT 0 CHECK (points >= 0);

-- Member codes are unique within a store (partial index ignores legacy NULLs).
CREATE UNIQUE INDEX IF NOT EXISTS uq_customers_store_member_code
    ON customers (store_id, member_code)
    WHERE member_code IS NOT NULL;

-- Backfill stable per-store codes for existing rows, ordered by signup.
WITH seq AS (
    SELECT id,
           'M' || LPAD(
               (ROW_NUMBER() OVER (PARTITION BY store_id ORDER BY created_at, id))::text,
               5, '0'
           ) AS code
    FROM customers
    WHERE member_code IS NULL
)
UPDATE customers c
SET member_code = seq.code
FROM seq
WHERE c.id = seq.id;
