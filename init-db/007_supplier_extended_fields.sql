-- Migration 007: Add extended contact and payment fields to suppliers

ALTER TABLE suppliers
    ADD COLUMN IF NOT EXISTS email          TEXT,
    ADD COLUMN IF NOT EXISTS line_id        TEXT,
    ADD COLUMN IF NOT EXISTS payment_method TEXT CHECK (payment_method IN ('promptpay', 'bank_account')),
    ADD COLUMN IF NOT EXISTS promptpay_number     TEXT,
    ADD COLUMN IF NOT EXISTS bank_name            TEXT,
    ADD COLUMN IF NOT EXISTS bank_account_number  TEXT,
    ADD COLUMN IF NOT EXISTS bank_account_name    TEXT,
    ADD COLUMN IF NOT EXISTS credit_days    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS logo_url       TEXT;
