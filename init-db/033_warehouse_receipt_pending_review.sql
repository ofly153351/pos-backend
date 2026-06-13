-- 033_warehouse_receipt_pending_review.sql
-- Approval workflow: add the 'pending_review' (Waiting Approval) state to the
-- warehouse receipt status lifecycle. Cashier/warehouse staff submit a draft for
-- approval (draft -> pending_review); an owner/manager then confirms it
-- (pending_review -> confirmed), which commits the stock movements.
-- Idempotent: drops and re-creates the status CHECK constraint widened.
ALTER TABLE warehouse_receipts DROP CONSTRAINT IF EXISTS warehouse_receipts_status_check;
ALTER TABLE warehouse_receipts
    ADD CONSTRAINT warehouse_receipts_status_check
    CHECK (status IN ('draft', 'pending_review', 'confirmed', 'cancelled'));
