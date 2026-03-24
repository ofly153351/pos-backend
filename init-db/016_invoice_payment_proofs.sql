ALTER TABLE invoice_payments
ADD COLUMN IF NOT EXISTS proof_url TEXT;

ALTER TABLE invoice_payments
ADD COLUMN IF NOT EXISTS proof_mime_type TEXT;

ALTER TABLE invoice_payments
ADD COLUMN IF NOT EXISTS proof_file_name TEXT;
