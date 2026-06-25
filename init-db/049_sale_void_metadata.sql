-- 049_sale_void_metadata.sql
-- Void/return audit trail on the sales table.

ALTER TABLE sales ADD COLUMN IF NOT EXISTS voided_at    TIMESTAMPTZ;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS voided_by    TEXT;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS void_reason  TEXT;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS void_type    TEXT;  -- 'void' | 'return'
