-- B13: a credit note must state why it was issued.
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS credit_reason TEXT;
