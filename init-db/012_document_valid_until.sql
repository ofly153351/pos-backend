-- Add valid_until for quotation documents
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS valid_until TIMESTAMPTZ;
