-- Add fax, email, website contact fields to stores
ALTER TABLE stores
    ADD COLUMN IF NOT EXISTS fax     TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS email   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS website TEXT NOT NULL DEFAULT '';
