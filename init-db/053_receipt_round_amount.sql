-- Add round_amount flag to store_receipt_settings.
-- Default TRUE: receipts show grand total rounded to nearest whole baht.
ALTER TABLE store_receipt_settings
  ADD COLUMN IF NOT EXISTS round_amount BOOLEAN NOT NULL DEFAULT TRUE;
