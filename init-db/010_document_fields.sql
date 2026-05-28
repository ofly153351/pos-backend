-- §86/4 compliance: customer address snapshot + item unit

ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS customer_address TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS customer_phone   TEXT NOT NULL DEFAULT '';

ALTER TABLE document_items
    ADD COLUMN IF NOT EXISTS unit VARCHAR(30) NOT NULL DEFAULT 'ชิ้น';
