-- Phase W4B follow-up — widen sales.status to the statuses the application actually writes.
--
-- The clean schema (001_schema.sql) declared the column inline as
--   status TEXT NOT NULL CHECK (status IN ('completed'))
-- so the only legal value was 'completed'. But the active reversal flow
-- (creditsale.Cancel → sale.SaleStatusVoided) writes 'voided', which failed with a
-- 23514 check-constraint violation on a clean database — i.e. cancellation was not
-- operational even though its location-restoration logic was correct.
--
-- Permitted set = EXACTLY the values the current code writes/reads for the sales table:
--   'completed'  — a created sale (sale.saleStatusCompleted)
--   'voided'     — a reversed sale, excluded from revenue/COGS (sale.SaleStatusVoided;
--                  also referenced by dashboard.saleVoidedStatus / finance.saleVoidedStatus)
-- No other status is written anywhere (credit_sales has its OWN status column with
-- pending/partial/completed/cancelled; that table is untouched here).
--
-- Additive + safe: every pre-existing sale row is 'completed' and still satisfies the
-- widened constraint, so no row is invalidated. Idempotent via DROP ... IF EXISTS +
-- re-ADD (same pattern as migration 033's status-CHECK widening).
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_status_check;
ALTER TABLE sales
    ADD CONSTRAINT sales_status_check
    CHECK (status IN ('completed', 'voided'));
