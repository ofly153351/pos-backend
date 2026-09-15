-- Payment snapshot fields for receipts generated from delivery orders.
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS payment_method TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS payment_reference TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS paid_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS change_amount NUMERIC(14,2) NOT NULL DEFAULT 0;