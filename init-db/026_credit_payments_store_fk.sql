-- 026: Add the missing foreign key on credit_payments.store_id.
--
-- credit_payments.store_id was an unconstrained TEXT column, so a payment could
-- reference a non-existent store. The service already scopes inserts, but the DB
-- had no defense-in-depth. 0 orphan rows verified before adding the constraint.
--
-- Idempotent: guarded so re-running (or running on a DB that already has it) is a
-- no-op.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_credit_payments_store'
          AND table_name = 'credit_payments'
    ) THEN
        ALTER TABLE credit_payments
            ADD CONSTRAINT fk_credit_payments_store
            FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE;
    END IF;
END $$;
