-- B05: record document payments in the existing invoice_payments ledger
-- (owner decision 2026-11-04: extend it, do not add a second ledger).
-- A row targets exactly one of invoice_id (legacy invoices) or document_id.
ALTER TABLE invoice_payments ALTER COLUMN invoice_id DROP NOT NULL;
ALTER TABLE invoice_payments
    ADD COLUMN IF NOT EXISTS document_id     TEXT REFERENCES documents(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS store_id        TEXT,
    ADD COLUMN IF NOT EXISTS created_by      TEXT,
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'invoice_payments_one_target') THEN
        ALTER TABLE invoice_payments
            ADD CONSTRAINT invoice_payments_one_target CHECK ((invoice_id IS NULL) <> (document_id IS NULL));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_invoice_payments_document ON invoice_payments(document_id) WHERE document_id IS NOT NULL;
-- Duplicate-request protection: one payment per (document, client key).
CREATE UNIQUE INDEX IF NOT EXISTS ux_invoice_payments_document_idem
    ON invoice_payments(document_id, idempotency_key)
    WHERE document_id IS NOT NULL AND idempotency_key IS NOT NULL;
