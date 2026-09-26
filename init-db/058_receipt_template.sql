-- Persist the selected A4 receipt layout so refresh/print/PDF keep the user's choice.
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS receipt_template INTEGER NOT NULL DEFAULT 1;

ALTER TABLE documents
    ADD CONSTRAINT documents_receipt_template_check
    CHECK (receipt_template IN (1, 2));
