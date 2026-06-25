-- 044: add is_active, is_default, updated_at to store_bank_accounts
-- Idempotent: ADD COLUMN IF NOT EXISTS
ALTER TABLE store_bank_accounts
  ADD COLUMN IF NOT EXISTS is_active  BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS is_default BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
