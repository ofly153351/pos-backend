-- Add Thai tax ID (เลขประจำตัวผู้เสียภาษี, 13 digits) to stores table
ALTER TABLE stores
    ADD COLUMN IF NOT EXISTS tax_id VARCHAR(13) NOT NULL DEFAULT '';
